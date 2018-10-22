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
	"net"
	"reflect"
	"time"

	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/internal/codec"
	"github.com/lonnng/nano/internal/message"
	"github.com/lonnng/nano/internal/packet"
	"github.com/lonnng/nano/session"
)

// Unhandled message buffer size
const packetBacklog = 1024
const funcBacklog = 1 << 8

var (
	// handler service singleton
	handler = newHandlerService()

	// serialized data
	hrd []byte // handshake response data
	hbd []byte // heartbeat packet data
)

func hbdEncode() {
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
		chLocalProcess chan unhandledMessage         // packets that process locally
		chCloseSession chan *session.Session         // closed session
		chFunction     chan func()                   // function that called in logic gorontine
		options        *options
	}

	unhandledMessage struct {
		agent   *agent
		lastMid uint
		handler reflect.Method
		args    []reflect.Value
	}
)

func newHandlerService() *handlerService {
	h := &handlerService{
		services:       make(map[string]*component.Service),
		handlers:       make(map[string]*component.Handler),
		chLocalProcess: make(chan unhandledMessage, packetBacklog),
		chCloseSession: make(chan *session.Session, packetBacklog),
		chFunction:     make(chan func(), funcBacklog),
		options:        &options{},
	}

	return h
}

// call handler with protected
func pcall(method reflect.Method, args []reflect.Value) {
	defer func() {
		if err := recover(); err != nil {
			logger.Println(fmt.Sprintf("nano/dispatch: %v", err))
			println(stack())
		}
	}()

	if r := method.Func.Call(args); len(r) > 0 {
		if err := r[0].Interface(); err != nil {
			logger.Println(err.(error).Error())
		}
	}
}

// call handler with protected
func pinvoke(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			logger.Println(fmt.Sprintf("nano/invoke: %v", err))
			println(stack())
		}
	}()

	fn()
}

func onSessionClosed(s *session.Session) {
	defer func() {
		if err := recover(); err != nil {
			logger.Println(fmt.Sprintf("nano/onSessionClosed: %v", err))
			println(stack())
		}
	}()

	session.Lifetime.Close(s)
}

// dispatch message to corresponding logic handler
func (h *handlerService) dispatch() {
	// close chLocalProcess & chCloseSession when application quit
	defer func() {
		if err := recover(); err != nil {
			logger.Println(fmt.Sprintf("Dispatcher exit unexpected: %v", err))
			println(stack())
		}
		close(h.chLocalProcess)
		close(h.chCloseSession)
		globalTicker.Stop()
		logger.Println("Main logic loop exit")
	}()

	// handle packet that sent to chLocalProcess
	for {
		select {
		case m := <-h.chLocalProcess: // logic dispatch
			m.agent.lastMid = m.lastMid
			pcall(m.handler, m.args)

		case s := <-h.chCloseSession: // session closed callback
			onSessionClosed(s)

		case fn := <-h.chFunction:
			pinvoke(fn)

		case <-globalTicker.C: // execute cron task
			cron()

		case t := <-timerManager.chCreatedTimer: // new timers
			timerManager.timers[t.id] = t

		case id := <-timerManager.chClosingTimer: // closing timers
			delete(timerManager.timers, id)

		case <-env.die: // application quit signal
			logger.Println("Got exit instruction, break logic loop")
			return
		}
	}
}

func (h *handlerService) register(comp component.Component, opts []component.Option) error {
	s := component.NewService(comp, opts)

	if _, ok := h.services[s.Name]; ok {
		return fmt.Errorf("handler: service already defined: %s", s.Name)
	}

	if err := s.ExtractHandler(); err != nil {
		return err
	}

	// register all handlers
	h.services[s.Name] = s
	for name, handler := range s.Handlers {
		h.handlers[fmt.Sprintf("%s.%s", s.Name, name)] = handler
	}
	return nil
}

func (h *handlerService) handle(conn net.Conn) {
	// create a client agent and startup write gorontine
	agent := newAgent(conn, h.options)

	// startup write goroutine
	go agent.write()

	if env.debug {
		logger.Println(fmt.Sprintf("New session established: %s", agent.String()))
	}

	// guarantee agent related resource be destroyed
	defer func() {
		agent.Close()
		if env.debug {
			logger.Println(fmt.Sprintf("Session read goroutine exit, SessionID=%d, UID=%d", agent.session.ID(), agent.session.UID()))
		}
	}()

	// read loop
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			logger.Println(fmt.Sprintf("Read message error: %s, session will be closed immediately", err.Error()))
			return
		}

		// TODO(warning): decoder use slice for performance, packet data should be copy before next Decode
		packets, err := agent.decoder.Decode(buf[:n])
		if err != nil {
			logger.Println(err.Error())
			return
		}

		if len(packets) < 1 {
			continue
		}

		// process all packet
		for i := range packets {
			if err := h.processPacket(agent, packets[i]); err != nil {
				logger.Println(err.Error())
				return
			}
		}
	}
}

func (h *handlerService) processPacket(agent *agent, p *packet.Packet) error {
	switch p.Type {
	case packet.Handshake:
		if _, err := agent.conn.Write(hrd); err != nil {
			return err
		}

		agent.setStatus(statusHandshake)
		if env.debug {
			logger.Println(fmt.Sprintf("Session handshake Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr()))
		}

	case packet.HandshakeAck:
		agent.setStatus(statusWorking)
		if env.debug {
			logger.Println(fmt.Sprintf("Receive handshake ACK Id=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr()))
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

func (h *handlerService) processMessage(agent *agent, msg *message.Message) {
	var lastMid uint
	switch msg.Type {
	case message.Request:
		lastMid = msg.ID
	case message.Notify:
		lastMid = 0
	}

	handler, ok := h.handlers[msg.Route]
	if !ok {
		logger.Println(fmt.Sprintf("nano/handler: %s not found(forgot registered?)", msg.Route))
		return
	}

	if pipe := h.options.pipeline; pipe != nil {
		err := pipe.Inbound().Process(agent.session, Message{msg})
		if err != nil {
			logger.Println("broken pipeline", err.Error())
			return
		}
	}

	var payload = msg.Data
	var data interface{}
	if handler.IsRawArg {
		data = payload
	} else {
		data = reflect.New(handler.Type.Elem()).Interface()
		err := serializer.Unmarshal(payload, data)
		if err != nil {
			logger.Println("deserialize error", err.Error())
			return
		}
	}

	if env.debug {
		logger.Println(fmt.Sprintf("UID=%d, Message={%s}, Data=%+v", agent.session.UID(), msg.String(), data))
	}

	args := []reflect.Value{handler.Receiver, agent.srv, reflect.ValueOf(data)}
	h.chLocalProcess <- unhandledMessage{agent, lastMid, handler.Method, args}
}

// DumpServices outputs all registered services
func (h *handlerService) DumpServices() {
	for name := range h.handlers {
		logger.Println("registered service", name)
	}
}
