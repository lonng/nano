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

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/codec"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/internal/message"
	"github.com/lonng/nano/internal/packet"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/scheduler"
)

var (
	// cached serialized data
	hrd []byte // handshake response data
	hbd []byte // heartbeat packet data
)

func cache() {
	data, err := json.Marshal(map[string]interface{}{
		"code": 200,
		"sys":  map[string]float64{"heartbeat": env.Heartbeat.Seconds()},
	})
	if err != nil {
		panic(err)
	}

	hrd, err = codec.Encode(packet.Handshake, data)
	if err != nil {
		panic(err)
	}

	hbd, err = codec.Encode(packet.Heartbeat, nil)
	if err != nil {
		panic(err)
	}
}

type LocalHandler struct {
	localServices  map[string]*component.Service // all registered service
	localHandlers  map[string]*component.Handler // all handler method
	remoteServices map[string][]*clusterpb.MemberInfo
	pipeline       pipeline.Pipeline
	rpcClient      *rpcClient
}

func NewHandler(pipeline pipeline.Pipeline) *LocalHandler {
	h := &LocalHandler{
		localServices:  make(map[string]*component.Service),
		localHandlers:  make(map[string]*component.Handler),
		remoteServices: map[string][]*clusterpb.MemberInfo{},
		pipeline:       pipeline,
	}

	return h
}

func (h *LocalHandler) register(comp component.Component, opts []component.Option) error {
	s := component.NewService(comp, opts)

	if _, ok := h.localServices[s.Name]; ok {
		return fmt.Errorf("handler: service already defined: %s", s.Name)
	}

	if err := s.ExtractHandler(); err != nil {
		return err
	}

	// register all localHandlers
	h.localServices[s.Name] = s
	for name, handler := range s.Handlers {
		h.localHandlers[fmt.Sprintf("%s.%s", s.Name, name)] = handler
	}
	return nil
}

func (h *LocalHandler) setRpcClient(client *rpcClient) {
	h.rpcClient = client
}

func (h *LocalHandler) initRemoteService(members []*clusterpb.MemberInfo) {
	for _, m := range members {
		h.addRemoteService(m)
	}
}

func (h *LocalHandler) addRemoteService(member *clusterpb.MemberInfo) {
	for _, s := range member.Services {
		h.remoteServices[s] = append(h.remoteServices[s], member)
	}
}

func (h *LocalHandler) LocalService() []string {
	var result []string
	for service := range h.localServices {
		result = append(result, service)
	}
	return result
}

func (h *LocalHandler) RemoteService() []string {
	var result []string
	for service := range h.remoteServices {
		result = append(result, service)
	}
	return result
}

func (h *LocalHandler) handle(conn net.Conn) {
	// create a client agent and startup write gorontine
	agent := newAgent(conn, h.pipeline)

	// startup write goroutine
	go agent.write()

	if env.Debug {
		log.Println(fmt.Sprintf("New session established: %s", agent.String()))
	}

	// guarantee agent related resource be destroyed
	defer func() {
		agent.Close()
		if env.Debug {
			log.Println(fmt.Sprintf("Session read goroutine exit, SessionID=%d, UID=%d", agent.session.ID(), agent.session.UID()))
		}
	}()

	// read loop
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(fmt.Sprintf("Read message error: %s, session will be closed immediately", err.Error()))
			return
		}

		// TODO(warning): decoder use slice for performance, packet data should be copy before next Decode
		packets, err := agent.decoder.Decode(buf[:n])
		if err != nil {
			log.Println(err.Error())
			return
		}

		if len(packets) < 1 {
			continue
		}

		// process all packet
		for i := range packets {
			if err := h.processPacket(agent, packets[i]); err != nil {
				log.Println(err.Error())
				return
			}
		}
	}
}

func (h *LocalHandler) processPacket(agent *agent, p *packet.Packet) error {
	switch p.Type {
	case packet.Handshake:
		if _, err := agent.conn.Write(hrd); err != nil {
			return err
		}

		agent.setStatus(statusHandshake)
		if env.Debug {
			log.Println(fmt.Sprintf("Session handshake Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr()))
		}

	case packet.HandshakeAck:
		agent.setStatus(statusWorking)
		if env.Debug {
			log.Println(fmt.Sprintf("Receive handshake ACK Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr()))
		}

	case packet.Data:
		if agent.status() < statusWorking {
			return fmt.Errorf("receive data on socket which not yet ACK, session will be closed immediately, remote=%s",
				agent.conn.RemoteAddr().String())
		}

		msg, err := message.Decode(p.Data)
		if err != nil {
			return err
		}
		h.processMessage(agent, msg)

	case packet.Heartbeat:
		// expected
	}

	agent.lastAt = time.Now().Unix()
	return nil
}

func (h *LocalHandler) remoteProcess(agent *agent, msg *message.Message) {
	index := strings.LastIndex(msg.Route, ".")
	if index < 0 {
		log.Println(fmt.Sprintf("nano/handler: invalid route %s", msg.Route))
		return
	}

	service := msg.Route[:index]
	members, found := h.remoteServices[service]
	if !found {
		log.Println(fmt.Sprintf("nano/handler: %s not found(forgot registered?)", msg.Route))
		return
	}
	// TODO: select remote
	conns, err := h.rpcClient.getConnArray(members[0].MemberAddr)
	if err != nil {
		log.Println(err)
		return
	}
	var data []byte
	if len(msg.Data) > 0 {
		data = make([]byte, len(msg.Data))
		copy(data, msg.Data)
	}
	client := clusterpb.NewMemberClient(conns.Get())
	switch msg.Type {
	case message.Request:
		request := &clusterpb.RequestMessage{
			Id:    msg.ID,
			Route: msg.Route,
			Data:  data,
		}
		_, err = client.HandleRequest(context.Background(), request)
	case message.Notify:
		request := &clusterpb.NotifyMessage{
			Route: msg.Route,
			Data:  data,
		}
		_, err = client.HandleNotify(context.Background(), request)
	}
	if err != nil {
		log.Println(fmt.Sprintf("Process remote message (%d:%s) error: %v", msg.ID, msg.Route, err))
		return
	}
}

func (h *LocalHandler) processMessage(agent *agent, msg *message.Message) {
	var lastMid uint64
	switch msg.Type {
	case message.Request:
		lastMid = msg.ID
	case message.Notify:
		lastMid = 0
	default:
		log.Println("Invalid message type: " + msg.Type.String())
		return
	}

	handler, found := h.localHandlers[msg.Route]
	if !found {
		h.remoteProcess(agent, msg)
		return
	} else {
		h.localProcess(handler, lastMid, agent, msg)
	}

}

func (h *LocalHandler) handleWS(conn *websocket.Conn) {
	c, err := newWSConn(conn)
	if err != nil {
		log.Println(err)
		return
	}
	h.handle(c)
}

func (h *LocalHandler) localProcess(handler *component.Handler, lastMid uint64, agent *agent, msg *message.Message) {
	if pipe := h.pipeline; pipe != nil {
		err := pipe.Inbound().Process(agent.session, msg)
		if err != nil {
			log.Println("Pipeline process failed: " + err.Error())
			return
		}
	}

	var payload = msg.Data
	var data interface{}
	if handler.IsRawArg {
		data = payload
	} else {
		data = reflect.New(handler.Type.Elem()).Interface()
		err := env.Serializer.Unmarshal(payload, data)
		if err != nil {
			log.Println("Deserialize failed: " + err.Error())
			return
		}
	}

	if env.Debug {
		log.Println(fmt.Sprintf("UID=%d, Message={%s}, Data=%+v", agent.session.UID(), msg.String(), data))
	}

	args := []reflect.Value{handler.Receiver, agent.srv, reflect.ValueOf(data)}
	scheduler.PushTask(func() {
		agent.lastMid = lastMid
		result := handler.Method.Func.Call(args)
		// TODO: send error message to client
		if len(result) > 0 {
			if err := result[0].Interface(); err != nil {
				log.Println(err.(error).Error())
			}
		}
	})
}
