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
	"context"
	"encoding/json"
	"fmt"
	originLog "log"
	"math/rand"
	"net"
	"reflect"
	"sort"
	"strings"
	"sync"
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
	"github.com/lonng/nano/serialize/msgpack"
	"github.com/lonng/nano/session"
	farmV1 "github.com/suhanyujie/throw_interface/golang_pb/farm/v1"
	throwV1 "github.com/suhanyujie/throw_interface/golang_pb/throw/v1"
	"google.golang.org/protobuf/proto"
)

var (
	// cached serialized data
	hrd []byte // handshake response data
	hbd []byte // heartbeat packet data
)

type rpcHandler func(session *session.Session, msg *message.Message, noCopy bool)

func cache() {
	hrdata := map[string]interface{}{
		"code": 200,
		"sys": map[string]interface{}{
			"heartbeat":  env.Heartbeat.Seconds(),
			"servertime": time.Now().UTC().Unix(),
		},
	}
	if dict, ok := message.GetDictionary(); ok {
		hrdata = map[string]interface{}{
			"code": 200,
			"sys": map[string]interface{}{
				"heartbeat":  env.Heartbeat.Seconds(),
				"servertime": time.Now().UTC().Unix(),
				"dict":       dict,
			},
		}
	}
	// data, err := json.Marshal(map[string]interface{}{
	// 	"code": 200,
	// 	"sys": map[string]float64{
	// 		"heartbeat": env.Heartbeat.Seconds(),
	// 	},
	// })
	data, err := json.Marshal(hrdata)
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
	localServices map[string]*component.Service // all registered service
	localHandlers map[string]*component.Handler // all handler method

	mu             sync.RWMutex
	remoteServices map[string][]*clusterpb.MemberInfo

	pipeline    pipeline.Pipeline
	currentNode *Node
}

func NewHandler(currentNode *Node, pipeline pipeline.Pipeline) *LocalHandler {
	h := &LocalHandler{
		localServices:  make(map[string]*component.Service),
		localHandlers:  make(map[string]*component.Handler),
		remoteServices: map[string][]*clusterpb.MemberInfo{},
		pipeline:       pipeline,
		currentNode:    currentNode,
	}

	return h
}

func (h *LocalHandler) register(comp component.Component, opts []component.Option) error {
	s := component.NewService(comp, opts)

	if _, ok := h.localServices[s.Name]; ok {
		return fmt.Errorf("handler: service already defined: %s", s.Name)
	}

	// 从 component 中提取 handler
	if err := s.ExtractHandler(); err != nil {
		return err
	}

	// register all localHandlers
	h.localServices[s.Name] = s
	for name, handler := range s.Handlers {
		n := fmt.Sprintf("%s.%s", s.Name, name)
		log.Println("Register local handler", n)
		h.localHandlers[n] = handler
	}
	return nil
}

func (h *LocalHandler) initRemoteService(members []*clusterpb.MemberInfo) {
	for _, m := range members {
		h.addRemoteService(m)
	}
}

func (h *LocalHandler) addRemoteService(member *clusterpb.MemberInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, s := range member.Services {
		log.Println("Register remote service", s)
		h.remoteServices[s] = append(h.remoteServices[s], member)
	}
}

func (h *LocalHandler) delMember(addr string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, members := range h.remoteServices {
		for i, maddr := range members {
			if addr == maddr.ServiceAddr {
				if i >= len(members)-1 {
					members = members[:i]
				} else {
					members = append(members[:i], members[i+1:]...)
				}
			}
		}
		if len(members) == 0 {
			delete(h.remoteServices, name)
		} else {
			h.remoteServices[name] = members
		}
	}
}

func (h *LocalHandler) LocalService() []string {
	var result []string
	for service := range h.localServices {
		result = append(result, service)
	}
	sort.Strings(result)
	return result
}

func (h *LocalHandler) RemoteService() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []string
	for service := range h.remoteServices {
		result = append(result, service)
	}
	sort.Strings(result)
	return result
}

func (h *LocalHandler) handle(conn net.Conn) {
	// create a client agent and startup write gorontine
	agent := newAgent(conn, h.pipeline, h.remoteProcess)
	h.currentNode.storeSession(agent.session)

	// startup write goroutine
	go agent.write()
	// originLog.Printf("[handle] env.debug: %v", env.Debug)
	if env.Debug {
		log.Println(fmt.Sprintf("[handle] New session established: %s", agent.String()))
	}

	// guarantee agent related resource be destroyed
	defer func() {
		request := &clusterpb.SessionClosedRequest{
			SessionId: agent.session.ID(),
		}

		members := h.currentNode.cluster.remoteAddrs()
		for _, remote := range members {
			log.Println("Notify remote server", remote)
			pool, err := h.currentNode.rpcClient.getConnPool(remote)
			if err != nil {
				log.Println("Cannot retrieve connection pool for address", remote, err)
				continue
			}
			client := clusterpb.NewMemberClient(pool.Get())
			_, err = client.SessionClosed(context.Background(), request)
			if err != nil {
				log.Println("Cannot closed session in remote address", remote, err)
				continue
			}
			if env.Debug {
				log.Println("Notify remote server success", remote)
			}
		}

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
			log.Println(fmt.Sprintf("[handle] Read message error: %s, session will be closed immediately", err.Error()))
			return
		}

		// TODO(warning): decoder use slice for performance, packet data should be copy before next Decode
		packets, err := agent.decoder.Decode(buf[:n])
		if err != nil {
			originLog.Printf("[handle] Decode err: %v", err)
			// process packets decoded
			//for _, p := range packets {
			//	if err := h.processPacket(agent, p); err != nil {
			//		log.Println(err.Error())
			//		return
			//	}
			//}
			return
		}

		// process all packets
		for _, p := range packets {
			if err := h.processPacket(agent, p); err != nil {
				log.Printf("[handle] processPacket err: %v", err)
				return
			}
		}
	}
}

func (h *LocalHandler) processPacket(agent *agent, p *packet.Packet) error {
	switch p.Type {
	case packet.Handshake:
		if err := env.HandshakeValidator(p.Data); err != nil {
			return err
		}

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
		// 因为定制化原因，只能接收数据帧，并解析出来
		// originLog.Printf("[processPacket] pack data: %v \n", p.Data)
		if len(p.Data) < 1 {
			originLog.Printf("[processPacket] pack data len is 0, maybe it's heartbeat")
			return nil
		}
		// 尝试解析为 特定对象
		var inputDataOri interface{}
		switch env.CustomProtocolStructType {
		case 2:
			inputDataOri = &farmV1.IRequest{}
			err := env.Serializer.Unmarshal(p.Data, inputDataOri)
			if err != nil {
				originLog.Printf("[processPacket] Unmarshal err: %v, data str: %s, data: %v\n", err, string(p.Data), p.Data)
				// 发送一个错误响应
				SendErrReply(agent, inputDataOri)
				return nil
			}
		default:
			inputDataOri = &throwV1.IRequestProtocol{}
			err := env.Serializer.Unmarshal(p.Data, inputDataOri)
			if err != nil {
				originLog.Printf("[processPacket] Unmarshal err: %v, data str: %s, data: %v\n", err, string(p.Data), p.Data)
				// 发送一个错误响应
				SendErrReply(agent, inputDataOri)
				return nil
			}
		}
		inputData := throwV1.IRequestProtocol{}
		switch env.CustomProtocolStructType {
		case 2:
			asserVal := inputDataOri.(*farmV1.IRequest)
			inputData.Action = asserVal.Action
			inputData.Method = asserVal.Method
			inputData.Callback = asserVal.Callback
			inputData.IsCompress = asserVal.IsCompress
			inputData.Data = asserVal.Data
		default:
			assertVal := inputDataOri.(*throwV1.IRequestProtocol)
			inputData = *assertVal
		}

		if inputData.Method == "UserLogin" || inputData.Method == "FarmUserLogin" || inputData.Method == "Reconnect" {
			// 表示登录
			agent.setStatus(statusWorking)
			if env.Debug {
				originLog.Printf("[processPacket] login sid=%d, Remote=%s", agent.session.ID(), agent.conn.RemoteAddr())
			}
		} else if inputData.Method == "HeartBeat" {
			switch env.CustomProtocolStructType {
			case 2:
				SendReply(agent, 1, &farmV1.NormalInfo{
					Msg: "heartbeat ok",
				}, &inputData)
			default:
				SendReply(agent, 1, &throwV1.DataInfoResp{
					Code: 0,
					Msg:  "heartbeat ok",
				}, &inputData)
			}
		} else {
			// if inputData.Data
			if agent.status() < statusWorking {
				originLog.Printf("[processPacket] conn status is less than statusWorking, user should login first...\n")
				SendErrReply(agent, &inputData)
				return nil
				// return fmt.Errorf("[processPacket] receive data on socket which not yet ACK, session will be closed immediately, remote=%s", agent.conn.RemoteAddr().String())
			}
		}
		// 将数据转换为 Message 对象
		msg, err := message.Decode(&inputData)
		if err != nil {
			originLog.Printf("[processPacket] message.Decode err: %v\n", err)
			return err
		}
		h.processMessage(agent, msg)

	case packet.Heartbeat:
		// expected
	}

	agent.lastAt = time.Now().Unix()
	return nil
}

// *throwV1.IRequestProtocol
func SendErrReply(agent *agent, req interface{}) {
	var data proto.Message
	if _, ok := req.(*throwV1.IRequestProtocol); ok {
		data = &throwV1.DataInfoResp{
			Code: -1,
			Msg:  "[SendErrReply] please login first...",
		}
	} else if _, ok := req.(*farmV1.IRequest); ok {
		data = &farmV1.NormalInfo{
			Msg: "[SendErrReply] please login first...",
		}
	}

	SendReply(agent, -1, data, req)
}

// req *throwV1.IRequestProtocol
func SendReply(agent *agent, code int32, data proto.Message, req interface{}) {
	var resp proto.Message
	msgPackCoder := msgpack.NewSerializer()
	dataBytes, _ := msgPackCoder.Marshal(data)
	callbackName := ""
	if reqVal, ok := req.(*throwV1.IRequestProtocol); ok {
		switch env.CustomProtocolStructType {
		case 2:
			resp = &farmV1.IResponse{
				Code:       code,
				IsCompress: true,
				Callback:   reqVal.Callback,
				Data:       dataBytes,
			}
		default:
			callbackName = fmt.Sprintf("%s_%s", reqVal.Action, reqVal.Method)
			resp = &throwV1.IResponseProtocol{
				Code:       code,
				IsCompress: true,
				Callback:   callbackName,
				Data:       dataBytes,
			}
		}
	}

	if err := agent.send(pendingMessage{
		typ:        message.Notify,
		route:      "error",
		mid:        agent.lastMid,
		payload:    dataBytes,
		payloadObj: resp,
	}); err != nil {
		originLog.Printf("[SendErrReply] send err: %v", err)
	}
}

func (h *LocalHandler) findMembers(service string) []*clusterpb.MemberInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.remoteServices[service]
}

func (h *LocalHandler) remoteProcess(session *session.Session, msg *message.Message, noCopy bool) {
	index := strings.LastIndex(msg.Route, ".")
	if index < 0 {
		log.Println(fmt.Sprintf("nano/handler: invalid route %s", msg.Route))
		return
	}

	service := msg.Route[:index]
	members := h.findMembers(service)
	if len(members) == 0 {
		log.Println(fmt.Sprintf("nano/handler: %s not found(forgot registered?)", msg.Route))
		return
	}

	// Select a remote service address
	// 1. Use the service address directly if the router contains binding item
	// 2. Select a remote service address randomly and bind to router
	var remoteAddr string
	if addr, found := session.Router().Find(service); found {
		remoteAddr = addr
	} else {
		remoteAddr = members[rand.Intn(len(members))].ServiceAddr
		session.Router().Bind(service, remoteAddr)
	}
	pool, err := h.currentNode.rpcClient.getConnPool(remoteAddr)
	if err != nil {
		log.Println(err)
		return
	}
	var data = msg.Data
	if !noCopy && len(msg.Data) > 0 {
		data = make([]byte, len(msg.Data))
		copy(data, msg.Data)
	}

	// Retrieve gate address and session id
	gateAddr := h.currentNode.ServiceAddr
	sessionId := session.ID()
	switch v := session.NetworkEntity().(type) {
	case *acceptor:
		gateAddr = v.gateAddr
		sessionId = v.sid
	}

	client := clusterpb.NewMemberClient(pool.Get())
	switch msg.Type {
	case message.Request:
		request := &clusterpb.RequestMessage{
			GateAddr:  gateAddr,
			SessionId: sessionId,
			Id:        msg.ID,
			Route:     msg.Route,
			Data:      data,
		}
		_, err = client.HandleRequest(context.Background(), request)
	case message.Notify:
		request := &clusterpb.NotifyMessage{
			GateAddr:  gateAddr,
			SessionId: sessionId,
			Route:     msg.Route,
			Data:      data,
		}
		_, err = client.HandleNotify(context.Background(), request)
	}
	if err != nil {
		log.Println(fmt.Sprintf("Process remote message (%d:%s) error: %+v", msg.ID, msg.Route, err))
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
	//defer func() {
	//	if err := recover(); err != nil {
	//		log.Println("[processMessage] panic err： %v", err)
	//	}
	//}()
	handler, found := h.localHandlers[msg.Route]
	if !found {
		originLog.Printf("[processMessage] coundnt found handler route: %s", msg.Route)
		h.remoteProcess(agent.session, msg, false)
	} else {
		h.localProcess(handler, lastMid, agent.session, msg)
	}
}

type WsConnWrapper struct {
	conn *websocket.Conn
	Data interface{}
}

func (h *LocalHandler) handleWS(connWrapper WsConnWrapper) {
	c, err := newWSConn(connWrapper.conn)
	if err != nil {
		originLog.Printf("[handleWS] ws conn err: %v", err)
		return
	}
	go h.handle(c)
}

func (h *LocalHandler) localProcess(handler *component.Handler, lastMid uint64, session *session.Session, msg *message.Message) {
	if pipe := h.pipeline; pipe != nil {
		err := pipe.Inbound().Process(session, msg)
		if err != nil {
			log.Println("[localProcess] Pipeline process failed: " + err.Error())
			return
		}
	}

	var payload = msg.DataOfPb
	var data interface{}
	if handler.IsRawArg {
		data = payload
	} else {
		// log.Println(fmt.Sprintf("[localProcess] handler.IsRawArg is false"))
		data = reflect.New(handler.Type.Elem()).Interface()
		err := msgpack.NewSerializer().Unmarshal(msg.Data, data)
		if err != nil {
			log.Println(fmt.Sprintf("[localProcess] Deserialize to %T failed: %+v (%v)", data, err, payload))
			return
		}
	}
	if env.Debug {
		// log.Println(fmt.Sprintf("[localProcess] UID=%d, Message={%s}, Data=%+v", session.UID(), msg.String(), data))
	}

	args := []reflect.Value{handler.Receiver, reflect.ValueOf(session), reflect.ValueOf(payload)}
	task := func() {
		switch v := session.NetworkEntity().(type) {
		case *agent:
			v.lastMid = lastMid
		case *acceptor:
			v.lastMid = lastMid
		}

		result := handler.Method.Func.Call(args)
		if len(result) > 0 {
			if err := result[0].Interface(); err != nil {
				log.Println(fmt.Sprintf("Service %s error: %+v", msg.Route, err))
			}
		}
	}

	index := strings.LastIndex(msg.Route, ".")
	if index < 0 {
		log.Println(fmt.Sprintf("[localProcess] nano/handler: invalid route %s", msg.Route))
		return
	}

	// A message can be dispatch to global thread or a user customized thread
	service := msg.Route[:index]
	if s, found := h.localServices[service]; found && s.SchedName != "" {
		log.Printf("[localProcess] scheduler: localServices")
		sched := session.Value(s.SchedName)
		if sched == nil {
			log.Println(fmt.Sprintf("[localProcess] nanl/handler: cannot found `schedular.LocalScheduler` by %s", s.SchedName))
			return
		}

		local, ok := sched.(scheduler.LocalScheduler)
		if !ok {
			log.Println(fmt.Sprintf("[localProcess] nanl/handler: Type %T does not implement the `schedular.LocalScheduler` interface",
				sched))
			return
		}
		local.Schedule(task)
	} else {
		log.Printf("[localProcess] scheduler: scheduler")
		scheduler.PushTask(task)
	}
}
