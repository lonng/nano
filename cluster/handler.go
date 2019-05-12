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
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/gorilla/websocket"
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

type (
	handler struct {
		services map[string]*component.Service // all registered service
		handlers map[string]*component.Handler // all handler method
		pipeline pipeline.Pipeline
	}
)

func newHandler(pipeline pipeline.Pipeline) *handler {
	h := &handler{
		services: make(map[string]*component.Service),
		handlers: make(map[string]*component.Handler),
		pipeline: pipeline,
	}

	return h
}

func (h *handler) register(comp component.Component, opts []component.Option) error {
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

func (h *handler) handle(conn net.Conn) {
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

func (h *handler) processPacket(agent *agent, p *packet.Packet) error {
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

func (h *handler) processMessage(agent *agent, msg *message.Message) {
	var lastMid uint
	switch msg.Type {
	case message.Request:
		lastMid = msg.ID
	case message.Notify:
		lastMid = 0
	}

	handler, ok := h.handlers[msg.Route]
	if !ok {
		log.Println(fmt.Sprintf("nano/handler: %s not found(forgot registered?)", msg.Route))
		return
	}

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

func (h *handler) handleWS(conn *websocket.Conn) {
	c, err := newWSConn(conn)
	if err != nil {
		log.Println(err)
		return
	}
	h.handle(c)
}
