package io

import (
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/lonnng/nano/codec"
	"github.com/lonnng/nano/message"
	"github.com/lonnng/nano/packet"
)

var (
	hsd []byte // handshake data
	had []byte // handshake ack data
)

func init() {
	var err error
	hsd, err = codec.Encode(packet.Handshake, nil)
	if err != nil {
		panic(err)
	}

	had, err = codec.Encode(packet.HandshakeAck, nil)
	if err != nil {
		panic(err)
	}
}

type (
	Callback func(data interface{})

	Connector struct {
		conn      net.Conn       // low-level connection
		codec     *codec.Decoder // decoder
		die       chan struct{}  // connector close channel
		chSend    chan []byte    // send queue
		mid       uint           // message id
		events    map[string]Callback
		responses map[uint]Callback

		connectedCallback func() // connected callback
	}
)

func NewConnector() *Connector {
	return &Connector{
		die:       make(chan struct{}),
		codec:     codec.NewDecoder(),
		chSend:    make(chan []byte, 64),
		mid:       1,
		events:    map[string]Callback{},
		responses: map[uint]Callback{},
	}
}

func (c *Connector) Start(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.conn = conn

	go c.write()

	// send handshake packet
	c.send(hsd)

	// read and process goroutine
	go c.read()

	return nil
}

func (c *Connector) OnConnected(callback func()) {
	c.connectedCallback = callback
}

func (c *Connector) Request(route string, v proto.Message, callback Callback) error {
	data, err := serialize(v)
	if err != nil {
		return err
	}

	msg := &message.Message{
		Type:  message.Request,
		Route: route,
		ID:    c.mid,
		Data:  data,
	}

	c.responses[c.mid] = callback
	if err := c.sendMessage(msg); err != nil {
		delete(c.responses, c.mid)
		return err
	}

	return nil
}

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

func (c *Connector) On(event string, callback Callback) {
	c.events[event] = callback
}

func (c *Connector) Close() {
	c.conn.Close()
	close(c.die)
}

func (c *Connector) sendMessage(msg *message.Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}

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
	buf := make([]byte, 512)

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
		cb, ok := c.events[msg.Route]
		if !ok {
			log.Println("event handler not found", msg.Route)
			return
		}

		cb(msg.Data)

	case message.Response:
		cb, ok := c.responses[msg.ID]
		if !ok {
			log.Println("response handler not found", msg.ID)
			return
		}

		cb(msg.Data)
		delete(c.responses, msg.ID)
	}
}

func serialize(v proto.Message) ([]byte, error) {
	data, err := proto.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}
