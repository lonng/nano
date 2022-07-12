package main

import (
	"encoding/json"
	"github.com/cute-angelia/go-utils/components/loggerV3"
	"github.com/cute-angelia/go-utils/syntax/ijson"
	"github.com/cute-angelia/go-utils/utils/conf"
	"github.com/lonng/nano/examples/demo/tankDemo/pb"
	"github.com/lonng/nano/internal/codec"
	"github.com/lonng/nano/internal/message"
	"github.com/lonng/nano/internal/packet"
	"github.com/lonng/nano/serialize/protobuf"
	"github.com/spf13/viper"
	"github.com/xtaci/kcp-go"
	"google.golang.org/protobuf/proto"
	"log"
	"net"
	"sync"
	"time"
)

var (
	hsd []byte // handshake data
	had []byte // handshake ack data
)

func init() {
	var err error
	hsd, err = codec.Encode(packet.Handshake, []byte(`{"sys":{"version":"1.1.1","type":"js-websocket"},"user":{"id":1001}}`))
	if err != nil {
		panic(err)
	}

	had, err = codec.Encode(packet.HandshakeAck, nil)
	if err != nil {
		panic(err)
	}
}

type (

	// Callback represents the callback type which will be called
	// when the correspond events is occurred.
	Callback func(data interface{})

	// Connector is a tiny Nano client
	Connector struct {
		conn   net.Conn       // low-level connection
		codec  *codec.Decoder // decoder
		die    chan struct{}  // connector close channel
		chSend chan []byte    // send queue
		mid    uint64         // message id

		// events handler
		muEvents sync.RWMutex
		events   map[string]Callback

		// response handler
		muResponses sync.RWMutex
		responses   map[uint64]Callback

		connectedCallback func() // connected callback
	}
)

// NewConnector create a new Connector
func NewConnector(conn net.Conn) *Connector {
	return &Connector{
		conn:      conn,
		die:       make(chan struct{}),
		codec:     codec.NewDecoder(),
		chSend:    make(chan []byte, 64),
		mid:       1,
		events:    map[string]Callback{},
		responses: map[uint64]Callback{},
	}
}

// Start connect to the server and send/recv between the c/s
func (c *Connector) Start() error {
	//conn, err := net.Dial("tcp", addr)
	//if err != nil {
	//	return err
	//}
	//
	//c.conn = conn

	go c.write()

	// send handshake packet
	c.send(hsd)

	// read and process network message
	go c.read()

	// call back
	c.events["onMembers"] = c.onMembers
	c.events["onNewUser"] = c.onNewUser

	return nil
}

// OnConnected set the callback which will be called when the client connected to the server
func (c *Connector) OnConnected(callback func()) {
	c.connectedCallback = callback
}

// Request send a request to server and register a callbck for the response
func (c *Connector) Request(route string, v proto.Message, callback Callback) error {
	data, err := serialize(v)
	if err != nil {
		log.Println(err)
		return err
	}

	msg := &message.Message{
		Type:  message.Request,
		Route: route,
		ID:    c.mid,
		Data:  data,
	}

	log.Println(route, msg)

	c.setResponseHandler(c.mid, callback)
	if err := c.sendMessage(msg); err != nil {
		c.setResponseHandler(c.mid, nil)
		return err
	}

	return nil
}

// Notify send a notification to server
func (c *Connector) Notify(route string, v proto.Message) error {
	data, err := serialize(v)
	if err != nil {
		return err
	}

	msg := &message.Message{
		Type:  message.Notify,
		Route: route,
		Data:  data,
	}
	return c.sendMessage(msg)
}

// On add the callback for the event
func (c *Connector) On(event string, callback Callback) {
	c.muEvents.Lock()
	defer c.muEvents.Unlock()

	c.events[event] = callback
}

// Close close the connection, and shutdown the benchmark
func (c *Connector) Close() {
	c.conn.Close()
	close(c.die)
}

func (c *Connector) eventHandler(event string) (Callback, bool) {
	c.muEvents.RLock()
	defer c.muEvents.RUnlock()

	cb, ok := c.events[event]
	return cb, ok
}

func (c *Connector) responseHandler(mid uint64) (Callback, bool) {
	c.muResponses.RLock()
	defer c.muResponses.RUnlock()

	cb, ok := c.responses[mid]
	return cb, ok
}

func (c *Connector) setResponseHandler(mid uint64, cb Callback) {
	c.muResponses.Lock()
	defer c.muResponses.Unlock()

	if cb == nil {
		delete(c.responses, mid)
	} else {
		c.responses[mid] = cb
	}
}

func (c *Connector) sendMessage(msg *message.Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}

	//log.Printf("%+v",msg)

	payload, err := codec.Encode(packet.Data, data)
	if err != nil {
		return err
	}

	c.mid++
	c.send(payload)

	return nil
}

func (c *Connector) write() {
	defer close(c.chSend)

	for {
		select {
		case data := <-c.chSend:
			if _, err := c.conn.Write(data); err != nil {
				log.Println(err.Error())
				c.Close()
			}

		case <-c.die:
			return
		}
	}
}

func (c *Connector) send(data []byte) {
	c.chSend <- data
}

func (c *Connector) read() {
	buf := make([]byte, 2048)

	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			log.Println(err.Error())
			c.Close()
			return
		}

		packets, err := c.codec.Decode(buf[:n])
		if err != nil {
			log.Println(err.Error())
			c.Close()
			return
		}

		for i := range packets {
			p := packets[i]
			c.processPacket(p)
		}
	}
}

func (c *Connector) processPacket(p *packet.Packet) {
	switch p.Type {
	case packet.Handshake:
		c.send(had)
		c.connectedCallback()
	case packet.Data:
		msg, err := message.Decode(p.Data)
		if err != nil {
			log.Println(err.Error())
			return
		}
		c.processMessage(msg)

	case packet.Kick:
		c.Close()
	}
}

func (c *Connector) processMessage(msg *message.Message) {
	switch msg.Type {
	case message.Push:
		cb, ok := c.eventHandler(msg.Route)
		if !ok {
			log.Println("event handler not found", msg.Route)
			return
		}

		cb(msg.Data)

	case message.Response:
		cb, ok := c.responseHandler(msg.ID)
		if !ok {
			log.Println("response handler not found", msg.ID)
			return
		}

		cb(msg.Data)
		c.setResponseHandler(msg.ID, nil)
	}
}

func (c *Connector) onMembers(msg interface{}) {
	// AllMembers contains all members uid
	type AllMembers struct {
		Members []int64 `json:"members"`
	}
	membs := AllMembers{}
	json.Unmarshal(msg.([]byte), &membs)
	log.Println(membs)
}

func (c *Connector) onNewUser(msg interface{}) {
	type NewUser struct {
		Content string `json:"content"`
	}
	membs := NewUser{}
	json.Unmarshal(msg.([]byte), &membs)
	log.Println(membs)
}

func serialize(v proto.Message) ([]byte, error) {
	data, err := proto.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func main() {
	// 加载日志
	conf.MustLoadConfigFile("./config.toml")

	// logger
	loggerV3.New(loggerV3.WithIsOnline(false), loggerV3.WithProject("chatKcpClient"))

	// 地址
	serverAddr := viper.GetString("common.server_addr")

	log.Println(serverAddr, "serverAddr")

	c, e := kcp.Dial(serverAddr)
	if nil != e {
		panic(e)
	}
	defer c.Close()

	log.Println("connecting...")

	connector := NewConnector(c)
	chReady := make(chan struct{})
	connector.connectedCallback = func() {
		log.Println("connected")
		chReady <- struct{}{}
	}
	connector.Start()

	<-chReady
	// 心跳
	go func() {
		heartByte, _ := codec.Encode(packet.Heartbeat, nil)
		d := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-d.C:
				// fmt.Println("The Current time is: ", tm)
				connector.send(heartByte)
			}
		}
	}()

	// 登陆
	connector.Request("Manager.Login", &pb.Login_Request{
		Uid: time.Now().Unix(),
	}, func(data interface{}) {
		datapb := pb.Login_Response{}
		protobuf.NewSerializer().Unmarshal(data.([]byte), &datapb)
		log.Println(ijson.Pretty(datapb))
	})

	// 加入房间
	connector.Request("RoomManager.CreateRoom", &pb.CreateRoom_Request{
		RoomId:         1,
		MaxPlayerCount: 2,
	}, func(data interface{}) {
		datapb := pb.CreateRoom_Response{}
		protobuf.NewSerializer().Unmarshal(data.([]byte), &datapb)
		log.Println(ijson.Pretty(datapb))
	})

	//msgJoin := message.New()
	//msgJoin.Route = "room.join"
	//msgJoin.Type = message.Request
	//msgJoin.ID = uint64(pb.ID_MSG_JoinRoom)
	//msgJoin.Data, _ = serialize(&pb.C2S_JoinRoomMsg{
	//	RoomId: proto.Uint64(1),
	//})
	//err := connector.sendMessage(msgJoin)

	// 发送消息
	//type UserMessage struct {
	//	Name    string `json:"name"`
	//	Content string `json:"content"`
	//}
	//userMsg := UserMessage{
	//	Name:    "UserMessage",
	//	Content: "xxxxx",
	//}
	//zz, _ := json.Marshal(userMsg)
	//msgJoin2 := message.New()
	//msgJoin2.Route = "room.message"
	//msgJoin2.Type = message.Request
	//msgJoin2.ID = uint64(time.Now().Unix())
	//msgJoin2.Data = zz
	//connector.sendMessage(msgJoin)

	wait := sync.WaitGroup{}
	wait.Add(1)
	wait.Wait()
}
