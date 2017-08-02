package packet

import (
	"errors"
	"fmt"
)

type PacketType byte

const (
	_            PacketType = iota
	Handshake               = 0x01 // packet for handshake request(client) <====> handshake response(server)
	HandshakeAck            = 0x02 // packet for handshake ack from client to server
	Heartbeat               = 0x03 // heartbeat packet
	Data                    = 0x04 // data packet
	Kick                    = 0x05 // disconnect message from server
)

var ErrWrongPacketType = errors.New("wrong packet type")

type Packet struct {
	Type   PacketType
	Length int
	Data   []byte
}

func New() *Packet {
	return &Packet{}
}

func (p *Packet) String() string {
	return fmt.Sprintf("Type: %d, Length: %d, Data: %s", p.Type, p.Length, string(p.Data))
}
