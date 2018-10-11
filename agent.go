// Copyright (c) nano Author. All Rights Reserved.
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

package nano

import (
	"errors"
	"fmt"
	// "net"
	"reflect"
	"sync/atomic"
	"time"

	// "github.com/jmesyan/nano/internal/codec"
	"github.com/jmesyan/nano/internal/message"
	// "github.com/jmesyan/nano/internal/packet"
	pb "github.com/jmesyan/nano/protos"
	"github.com/jmesyan/nano/session"
	"sync"
)

const (
	agentWriteBacklog = 16
	cmdAck            = 0x8000000
)

var (
	// ErrBrokenPipe represents the low-level connection has broken.
	ErrBrokenPipe = errors.New("broken low-level pipe")
	// ErrBufferExceed indicates that the current session buffer is full and
	// can not receive more data.
	ErrBufferExceed = errors.New("session send buffer exceed")

	agentManager = make(map[int32]*agent)
	agentLock    sync.Mutex
)

type (
	// Agent corresponding a user, used for store raw conn information
	agent struct {
		sync.Mutex
		// regular agent member
		cid     int32                         //channelid
		cmd     int32                         //cmd
		typ     int32                         //tick
		mid     int32                         //message id
		route   string                        //route
		session *session.Session              // session
		conn    pb.GrpcService_MServiceClient // low-level conn fd
		lastMid int32                         // last message id
		state   int32                         // current agent state
		chDie   chan struct{}                 // wait for close
		chSend  chan pendingMessage           // push message queue
		lastAt  int64                         // last heartbeat unix time stamp
		options *options                      // options

		srv reflect.Value // cached session reflect.Value
	}

	pendingMessage struct {
		cid     int32
		cmd     int32
		typ     int32       // message type
		route   string      // message route(push)
		mid     int32       // response message id(response)
		payload interface{} // payload
	}
)

//get the agent
func chanAgent(conn pb.GrpcService_MServiceClient, message *pb.GrpcMessage, options *options) *agent {
	cid := message.Cid
	agent, ok := agentManager[cid]
	if !ok {
		agentLock.Lock()
		agentLock.Unlock()
		agent = newAgent(cid, options)
		go agent.write()
		agentManager[cid] = agent

	}
	agent.conn = conn
	agent.cmd = message.Cmd
	agent.typ = message.Type
	agent.mid = message.Mid
	agent.route = message.Route
	return agent
}

// Create new agent instance
func newAgent(cid int32, options *options) *agent {
	a := &agent{
		state:   statusStart,
		chDie:   make(chan struct{}),
		lastAt:  time.Now().Unix(),
		chSend:  make(chan pendingMessage, agentWriteBacklog),
		options: options,
		cid:     cid,
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
	a.chSend <- m
	return
}

func (a *agent) LastMid() int32 {
	return a.lastMid
}

func (a *agent) CID() int32 {
	return a.cid
}

func (a *agent) CMD() int32 {
	return a.cmd
}

func (a *agent) MID() int32 {
	return a.mid
}

func (a *agent) Route() string {
	return a.route
}

// Push, implementation for session.NetworkEntity interface
func (a *agent) Push(route string, cmd int32, v interface{}) error {
	return a.PushChannel(a.cid, route, cmd, v)
}

func (a *agent) PushChannel(cid int32, route string, cmd int32, v interface{}) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	if env.debug {
		switch d := v.(type) {
		case []byte:
			logger.Println(fmt.Sprintf("CID=%d, CMD=%d,Type=Push, ID=%d, UID=%d, Route=%s, Data=%dbytes", cid,
				cmd, a.session.ID(), a.session.UID(), route, len(d)))
		default:
			logger.Println(fmt.Sprintf("CID=%d, CMD=%d,Type=Push, ID=%d, UID=%d, Route=%s, Data=%+v", cid,
				cmd, a.session.ID(), a.session.UID(), route, v))
		}
	}
	if cmd > 0 {
		cmd = cmd | cmdAck
	}
	return a.send(pendingMessage{cid: cid, cmd: cmd, typ: message.Push, route: route, payload: v})
}

// Response, implementation for session.NetworkEntity interface
// Response message to session
func (a *agent) Response(cmd int32, v interface{}) error {
	return a.ResponseMID(a.mid, cmd, v)
}

// Response, implementation for sssion.NetworkEntity interface
// Response message to session
func (a *agent) ResponseMID(mid int32, cmd int32, v interface{}) error {
	if a.status() == statusClosed {
		return ErrBrokenPipe
	}

	if mid <= 0 {
		return ErrSessionOnNotify
	}

	if len(a.chSend) >= agentWriteBacklog {
		return ErrBufferExceed
	}

	if env.debug {
		switch d := v.(type) {
		case []byte:
			logger.Println(fmt.Sprintf("CID=%d, CMD=%d,Type=Response, ID=%d, UID=%d, Mid=%d, Data=%dbytes", a.CID(),
				cmd, a.session.ID(), a.session.UID(), mid, len(d)))
		default:
			logger.Println(fmt.Sprintf("CID=%d, CMD=%d,Type=Response, ID=%d, UID=%d, Mid=%d, Data=%+v", a.CID(),
				cmd, a.session.ID(), a.session.UID(), mid, v))
		}
	}
	if cmd > 0 {
		cmd = cmd | cmdAck
	}
	return a.send(pendingMessage{cid: a.cid, cmd: cmd, typ: message.Response, mid: mid, payload: v})
}

// Close, implementation for session.NetworkEntity interface
// Close closes the agent, clean inner state and close low-level connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (a *agent) Close() error {
	if a.status() == statusClosed {
		return ErrCloseClosedSession
	}
	a.setStatus(statusClosed)

	if env.debug {
		logger.Println(fmt.Sprintf("Session closed, ID=%d, UID=%d",
			a.session.ID(), a.session.UID()))
	}

	// prevent closing closed channel
	select {
	case <-a.chDie:
		// expect
	default:
		if !isReconnecting() {
			close(a.chDie)
			// handler.chCloseSession <- a.session
		}
	}
	logger.Println("the agent will close")
	return a.conn.CloseSend()
}

// RemoteAddr, implementation for session.NetworkEntity interface
// returns the remote network address.
// func (a *agent) RemoteAddr() net.Addr {
// 	return a.conn.RemoteAddr()
// }

// String, implementation for Stringer interface
func (a *agent) String() string {
	return fmt.Sprintf("LastTime=%d", a.lastAt)
}

func (a *agent) status() int32 {
	return atomic.LoadInt32(&a.state)
}

func (a *agent) setStatus(state int32) {
	atomic.StoreInt32(&a.state, state)
}

func (a *agent) write() {
	// ticker := time.NewTicker(env.heartbeat)
	chWrite := make(chan *pb.GrpcMessage, agentWriteBacklog)
	// clean func
	defer func() {
		ticker.Stop()
		close(a.chSend)
		close(chWrite)
		a.Close()
		if env.debug {
			logger.Println(fmt.Sprintf("Session write goroutine exit, SessionID=%d, UID=%d", a.session.ID(), a.session.UID()))
		}
	}()

	for {
		select {
		case <-ticker.C:
			// deadline := time.Now().Add(-2 * env.heartbeat).Unix()
			// if a.lastAt < deadline {
			// 	logger.Println(fmt.Sprintf("Session heartbeat timeout, LastTime=%d, Deadline=%d", a.lastAt, deadline))
			// 	return
			// }
			handler.heartbeatTimeoutCb()
			// chWrite <- hbd
		case attempts := <-reconnect.attempts:
			logger.Println("come the reconnect attempts", attempts)
			if attempts > 0 && attempts <= reconnect.reconnectMaxAttempts {
				logger.Println(fmt.Printf("第%d次尝试重连：", attempts))
				reconnect.reconnectAttempts = attempts
				reconnect.trying = true
				caddr, err := net.ResolveTCPAddr("tcp", reconnect.addr)
				if err != nil {
					logger.Println(err.Error())
				} else {
					conn, err := net.DialTCP("tcp", nil, caddr)
					if err != nil {
						logger.Println(err.Error())
					} else {
						logger.Println(fmt.Printf("第%d次尝试重连成功", attempts))
						a.conn = conn
						conn.SetNoDelay(true)
						go handler.handleC(a, conn)

					}
				}
				reconnect.trying = false
			}
		case data := <-chWrite:
			// close agent while low-level conn broken
			if err := a.conn.SendMsg(data); err != nil {
				logger.Println(err.Error())
				return
			}

		case data := <-a.chSend:
			payload, err := serializeOrRaw(data.payload)
			if err != nil {
				logger.Println(err.Error())
				break
			}

			// construct message and encode
			m := &pb.GrpcMessage{
				Cid:   data.cid,
				Cmd:   data.cmd,
				Type:  data.typ,
				Mid:   data.mid,
				Route: data.route,
				Data:  payload,
			}
			if pipe := a.options.pipeline; pipe != nil {
				err := pipe.Outbound().Process(a.session, *m)
				if err != nil {
					logger.Println("broken pipeline", err.Error())
					break
				}
			}

			// em, err := m.Encode()
			// if err != nil {
			// 	logger.Println(err.Error())
			// 	break
			// }

			// packet encode
			// p, err := codec.Encode(packet.Data, m)
			// if err != nil {
			// 	logger.Println(err)
			// 	break
			// }
			chWrite <- m

		case <-a.chDie: // agent closed signal
			return

		case <-env.die: // application quit
			return
		}
	}
}
