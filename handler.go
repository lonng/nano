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
	"encoding/json"
	"fmt"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/lonnng/nano/codec"
	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/message"
	"github.com/lonnng/nano/packet"
	"github.com/lonnng/nano/session"
)

// Unhandled message buffer size
const packetBacklog = 1024

var (
	// handler service singleton
	handler = newHandlerService()

	// serialized data
	hrd []byte // handshake response data
	hbd []byte // heartbeat packet data
)

func init() {
	data, err := json.Marshal(map[string]interface{}{
		"code": 200,
		"sys":  map[string]float64{"heartbeat": env.heartbeat.Seconds()},
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

type (
	handlerService struct {
		services       map[string]*component.Service // all registered service
		handlers       map[string]*component.Handler // all handler method
		chLocalProcess chan *unhandledMessage        // packets that process locally
		chCloseSession chan *session.Session         // closed session
	}

	unhandledMessage struct {
		handler reflect.Method
		args    []reflect.Value
	}
)

func newHandlerService() *handlerService {
	h := &handlerService{
		services:       make(map[string]*component.Service),
		handlers:       make(map[string]*component.Handler),
		chLocalProcess: make(chan *unhandledMessage, packetBacklog),
		chCloseSession: make(chan *session.Session, packetBacklog),
	}

	// startup logic dispatcher
	go h.dispatch()

	return h
}

// call handler with protected
func pcall(method reflect.Method, args []reflect.Value) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(fmt.Sprintf("nano/dispatch: Error=%+v, Stack=%s", err, stack()))
		}
	}()

	if r := method.Func.Call(args); len(r) > 0 {
		if err := r[0].Interface(); err != nil {
			log.Println(err.(error).Error())
		}
	}
}

func onSessionClosed(s *session.Session) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(fmt.Sprintf("nano/onSessionClosed: Error=%+v, Stack=%s", err, stack()))
		}
	}()

	env.muCallbacks.RLock()
	defer env.muCallbacks.RUnlock()

	for _, fn := range env.callbacks {
		fn(s)
	}
}

// dispatch message to corresponding logic handler
func (h *handlerService) dispatch() {
	// close chLocalProcess & chCloseSession when application quit
	defer func() {
		close(h.chLocalProcess)
		close(h.chCloseSession)
	}()

	// handle packet that sent to chLocalProcess
	for {
		select {
		case m := <-h.chLocalProcess: // logic dispatch
			pcall(m.handler, m.args)

		case s := <-h.chCloseSession: // session closed callback
			onSessionClosed(s)

		case <-env.die: // application quit signal
			return
		}
	}
}

func (h *handlerService) register(receiver component.Component) error {
	s := &component.Service{
		Type:     reflect.TypeOf(receiver),
		Receiver: reflect.ValueOf(receiver),
	}
	s.Name = reflect.Indirect(s.Receiver).Type().Name()

	if _, ok := h.services[s.Name]; ok {
		return fmt.Errorf("handler: service already defined: %s", s.Name)
	}

	if err := s.ScanHandler(); err != nil {
		return err
	}

	// register all handlers
	h.services[s.Name] = s
	for name, method := range s.Methods {
		h.handlers[fmt.Sprintf("%s.%s", s.Name, name)] = method
	}
	return nil
}

func (h *handlerService) handle(conn net.Conn) {
	// create a client agent and startup write gorontine
	agent := newAgent(conn)
	go agent.write()

	if env.debug {
		log.Println(fmt.Sprintf("New session established: %s", agent.String()))
	}

	// read loop
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(fmt.Sprintf("Read message error: %s, session will be closed immediately", err.Error()))
			return
		}

		// TODO(warning): codec use slice for performance, packet data should be copy before next Decode
		packets, err := agent.codec.Decode(buf[:n])
		if err != nil {
			log.Println(err.Error())
			return
		}

		// process all packet
		for i := range packets {
			h.processPacket(agent, packets[i])
		}
	}
}

func (h *handlerService) processPacket(agent *agent, p *packet.Packet) {
	switch p.Type {
	case packet.Handshake:
		if _, err := agent.socket.Write(hrd); err != nil {
			log.Println(err.Error())
			agent.Close()
		}

		if env.debug {
			log.Println(fmt.Sprintf("Session handshake Id=%d, Remote=%s", agent.session.ID, agent.socket.RemoteAddr()))
		}
		agent.setStatus(statusHandshake)

	case packet.HandshakeAck:
		agent.setStatus(statusWorking)

		if env.debug {
			log.Println(fmt.Sprintf("Receive handshake ACK Id=%d, Remote=%s", agent.session.ID, agent.socket.RemoteAddr()))
		}

	case packet.Data:
		if agent.status() < statusWorking {
			log.Println(fmt.Sprintf("receive data on socket which not yet ACK, remote=%s", agent.socket.RemoteAddr().String()))
			agent.Close()
			return
		}

		msg, err := message.Decode(p.Data)
		if err != nil {
			log.Println(err.Error())
			return
		}
		h.processMessage(agent.session, msg)
		fallthrough

	case packet.Heartbeat:
		// expected
	}

	agent.lastAt = time.Now().Unix()
}

func (h *handlerService) processMessage(session *session.Session, msg *message.Message) {
	switch msg.Type {
	case message.Request:
		session.LastRID = msg.ID
	case message.Notify:
		session.LastRID = 0
	}

	handler, ok := h.handlers[msg.Route]
	if !ok {
		log.Println(fmt.Sprintf("nano/handler: %s not found(forgot registered?)", msg.Route))
		return
	}

	var data interface{}
	if handler.IsRawArg {
		data = msg.Data
	} else {
		data = reflect.New(handler.Type.Elem()).Interface()
		err := serializer.Deserialize(msg.Data, data)
		if err != nil {
			log.Println("deserialize error", err.Error())
			return
		}
	}

	if env.debug {
		log.Println(fmt.Sprintf("Uid=%d, Message={%s}, Data=%+v", session.Uid, msg.String(), data))
	}

	args := []reflect.Value{handler.Receiver, reflect.ValueOf(session), reflect.ValueOf(data)}
	h.chLocalProcess <- &unhandledMessage{handler.Method, args}
}

func (h *handlerService) dumpServiceMap() {
	for name := range h.handlers {
		log.Println("registered service", name)
	}
}
