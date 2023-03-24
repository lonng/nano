// Copyright (c) nano Authors. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cluster

import (
	"errors"
	"fmt"
	originLog "log"
	"net"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/lonng/nano/pkg/utils/jsonx"

	"google.golang.org/protobuf/proto"

	"github.com/lonng/nano/internal/codec"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/internal/message"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	throwV1 "github.com/suhanyujie/throw_interface/golang_pb/throw/v1"
)

const (
	agentWriteBacklog = 160000 // 16
)

var (
	// ErrBrokenPipe represents the low-level connection has broken.
	ErrBrokenPipe = errors.New("broken low-level pipe")
	// ErrBufferExceed indicates that the current session buffer is full and
	// can not receive more data.
	ErrBufferExceed = errors.New("session send buffer exceed")
)

type (
	// Agent corresponding a user, used for store raw conn information
	agent struct {
		// regular agent member
		session  *session.Session    // session
		conn     net.Conn            // low-level conn fd
		lastMid  uint64              // last message id
		state    int32               // current agent state
		chDie    chan struct{}       // wait for close
		chSend   chan pendingMessage // push message queue
		lastAt   int64               // last heartbeat unix time stamp
		decoder  *codec.Decoder      // binary decoder
		pipeline pipeline.Pipeline

		rpcHandler rpcHandler
		srv        reflect.Value // cached session reflect.Value
	}

	pendingMessage struct {
		typ        message.Type // message type
		route      string       // message route(push)
		mid        uint64       // response message id(response)
		payload    interface{}  // payload
		payloadObj proto.Message
	}
)

// Create new agent instance
func newAgent(conn net.Conn, pipeline pipeline.Pipeline, rpcHandler rpcHandler) *agent {
	a := &agent{
		conn:       conn,
		state:      statusStart,
		chDie:      make(chan struct{}),
		lastAt:     time.Now().Unix(),
		chSend:     make(chan pendingMessage, agentWriteBacklog),
		decoder:    codec.NewDecoder(),
		pipeline:   pipeline,
		rpcHandler: rpcHandler,
	}

	// binding session
	s := session.New(a)
	a.session = s
	a.srv = reflect.ValueOf(s)

	return a
}

func (a *agent) send(m pendingMessage) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrBrokenPipe
		}
	}()
	// 将 pendingMessage 消息进行转换
	a.chSend <- m
	return
}

// LastMid implements the session.NetworkEntity interface
func (a *agent) LastMid() uint64 {
	return a.lastMid
}

// Push, implementation for session.NetworkEntity interface
func (a *agent) Push(route string, v interface{}) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	if env.Debug {
		//switch d := v.(type) {
		//case []byte:
		//	log.Println(fmt.Sprintf("[Push] sid=%d, uid=%d, Data=%dbytes", a.session.ID(), a.session.UID(), len(d)))
		//default:
		//	log.Println(fmt.Sprintf("[Push] sid=%d, uid=%d,Data=%+v", a.session.ID(), a.session.UID(), v))
		//}
	}
	pm := pendingMessage{typ: message.Push, route: route, payload: v}
	if val, ok := v.(proto.Message); ok {
		pm.payloadObj = val
	}

	return a.send(pm)
}

// RPC, implementation for session.NetworkEntity interface
func (a *agent) RPC(route string, v interface{}) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	// TODO: buffer
	data, err := message.Serialize(v)
	if err != nil {
		return err
	}
	msg := &message.Message{
		Type:  message.Notify,
		Route: route,
		Data:  data,
	}
	a.rpcHandler(a.session, msg, true)
	return nil
}

// Response, implementation for session.NetworkEntity interface
// Response message to session
func (a *agent) Response(v interface{}) error {
	return a.ResponseMid(a.lastMid, v)
}

// ResponseMid, implementation for session.NetworkEntity interface
// Response message to session
func (a *agent) ResponseMid(mid uint64, v interface{}) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if mid <= 0 {
		return ErrSessionOnNotify
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	if env.Debug {
		switch d := v.(type) {
		case []byte:
			log.Println(fmt.Sprintf("[ResponseMid] Type=Response, ID=%d, UID=%d, MID=%d, Data=%dbytes",
				a.session.ID(), a.session.UID(), mid, len(d)))
		default:
			// 尝试解析
			if obj, ok := v.(*throwV1.IResponseProtocol); ok {
				dataObj := &throwV1.DataInfoResp{}
				err := env.Serializer.Unmarshal(obj.Data, dataObj)
				if err == nil {
					log.Println(fmt.Sprintf("[ResponseMid] Type=Response, ID=%d, UID=%d, MID=%d, respData: %s",
						a.session.ID(), a.session.UID(), mid, jsonx.ToJsonIgnoreErr(dataObj)))
				}
			} else {
				log.Println(fmt.Sprintf("Type=Response, ID=%d, UID=%d, MID=%d, Data=%+v",
					a.session.ID(), a.session.UID(), mid, v))
			}
		}
	}
	pm := pendingMessage{typ: message.Response, mid: mid, payload: v}
	if val, ok := v.(proto.Message); ok {
		pm.payloadObj = val
	}
	if err := a.send(pm); err != nil {
		originLog.Printf("[ResponseMid] send err: %v\n", err)
	}

	return nil
}

// Close, implementation for session.NetworkEntity interface
// Close closes the agent, clean inner state and close low-level connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (a *agent) Close() error {
	if a.status() == statusClosed {
		return ErrCloseClosedSession
	}
	a.setStatus(statusClosed)

	if env.Debug {
		log.Println(fmt.Sprintf("Session closed, ID=%d, UID=%d, IP=%s",
			a.session.ID(), a.session.UID(), a.conn.RemoteAddr()))
	}

	// prevent closing closed channel
	select {
	case <-a.chDie:
		// expect
	default:
		close(a.chDie)
		scheduler.PushTask(func() { session.Lifetime.Close(a.session) })
	}

	return a.conn.Close()
}

// RemoteAddr, implementation for session.NetworkEntity interface
// returns the remote network address.
func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

// String, implementation for Stringer interface
func (a *agent) String() string {
	return fmt.Sprintf("Remote=%s, LastTime=%d", a.conn.RemoteAddr().String(), atomic.LoadInt64(&a.lastAt))
}

func (a *agent) status() int32 {
	return atomic.LoadInt32(&a.state)
}

func (a *agent) setStatus(state int32) {
	atomic.StoreInt32(&a.state, state)
}

func (a *agent) write() {
	ticker := time.NewTicker(env.Heartbeat)
	chWrite := make(chan []byte, agentWriteBacklog)
	// clean func
	defer func() {
		ticker.Stop()
		close(a.chSend)
		close(chWrite)
		a.Close()
		if env.Debug {
			log.Println(fmt.Sprintf("Session write goroutine exit, SessionID=%d, UID=%d", a.session.ID(), a.session.UID()))
		}
	}()

	for {
		select {
		case <-ticker.C:
			deadline := time.Now().Add(-2 * env.Heartbeat).Unix()
			if atomic.LoadInt64(&a.lastAt) < deadline {
				log.Println(fmt.Sprintf("Session heartbeat timeout, LastTime=%d, Deadline=%d", atomic.LoadInt64(&a.lastAt), deadline))
				return
			}
			chWrite <- hbd

		case data := <-chWrite:
			// close agent while low-level conn broken
			if _, err := a.conn.Write(data); err != nil {
				log.Println(err.Error())
				return
			}

		case data := <-a.chSend:
			// 检查是否是 proto 结构
			dataForProto, ok := data.payloadObj.(proto.Message)
			if !ok {
				originLog.Printf("[write] payload is not proto struct")
				continue
			}
			dataBytes, err := env.Serializer.Marshal(dataForProto)
			if err != nil {
				originLog.Printf("[write] Serializer.Marshal err: %v", err)
				continue
			}
			// construct message and encode
			m := &message.Message{
				Type: data.typ,
				Data: dataBytes, // payload
				// DataOfPb: dataForProto,
				Route: data.route,
				ID:    data.mid,
			}
			if pipe := a.pipeline; pipe != nil {
				err := pipe.Outbound().Process(a.session, m)
				if err != nil {
					log.Println("broken pipeline", err.Error())
					break
				}
			}

			p := m.Data
			//p, err := m.Encode()
			//if err != nil {
			//	log.Println(err.Error())
			//	break
			//}

			// packet encode 这里不再需要了
			//p, err := codec.Encode(packet.Data, em)
			//if err != nil {
			//	log.Println(err)
			//	break
			//}
			chWrite <- p

		case <-a.chDie: // agent closed signal
			return

		case <-env.Die: // application quit
			return
		}
	}
}
