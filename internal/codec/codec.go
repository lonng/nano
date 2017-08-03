package codec

import (
	"bytes"
	"errors"

	"github.com/lonnng/nano/internal/packet"
)

const HeadLength = 4
const MaxPacketSize = 64 * 1024

var ErrPacketSizeExcced = errors.New("packet size exceed")

type Decoder struct {
	*bytes.Buffer
	size int  // last packet length
	typ  byte // last packet type
}

func NewDecoder() *Decoder {
	return &Decoder{Buffer: bytes.NewBuffer(nil), size: -1}
}

// TODO(Warning): shared slice
func (c *Decoder) Decode(data []byte) (packets []*packet.Packet, err error) {
	c.Write(data)

	// check length
	if c.Len() < HeadLength {
		return
	}

	// first time
	if c.size < 0 {
		header := c.Next(HeadLength)
		c.typ = header[0]
		c.size = bytesToInt(header[1:])
		if c.typ < packet.Handshake || c.typ > packet.Kick {
			return packets, packet.ErrWrongPacketType
		}
	}

	// packet length limitation
	if c.size > MaxPacketSize {
		return packets, ErrPacketSizeExcced
	}

	for c.size <= c.Len() {
		p := &packet.Packet{Type: packet.PacketType(c.typ), Length: c.size, Data: c.Next(c.size)}
		packets = append(packets, p)

		// more packet
		if c.Len() < HeadLength {
			c.size = -1
			break
		} else {
			header := c.Next(HeadLength)
			c.typ = header[0]
			c.size = bytesToInt(header[1:])
			if c.typ < packet.Handshake || c.typ > packet.Kick {
				return packets, packet.ErrWrongPacketType
			}

			if c.size > MaxPacketSize {
				return packets, ErrPacketSizeExcced
			}

		}
	}

	return packets, nil
}

// Protocol refs: https://github.com/NetEase/pomelo/wiki/Communication-Protocol
//
// -<type>-|--------<length>--------|-<data>-
// --------|------------------------|--------
// 1 byte packet type, 3 bytes packet data length(big end), and data segment
func Encode(typ packet.PacketType, data []byte) ([]byte, error) {
	if typ < packet.Handshake || typ > packet.Kick {
		return nil, packet.ErrWrongPacketType
	}

	p := &packet.Packet{Type: typ, Length: len(data)}
	buf := make([]byte, p.Length+HeadLength)
	buf[0] = byte(p.Type)

	copy(buf[1:HeadLength], intToBytes(p.Length))
	copy(buf[HeadLength:], data)

	return buf, nil
}

// Decode packet data length byte to int(Big end)
func bytesToInt(b []byte) int {
	result := 0
	for _, v := range b {
		result = result<<8 + int(v)
	}
	return result
}

// Encode packet data length to bytes(Big end)
func intToBytes(n int) []byte {
	buf := make([]byte, 3)
	buf[0] = byte((n >> 16) & 0xFF)
	buf[1] = byte((n >> 8) & 0xFF)
	buf[2] = byte(n & 0xFF)
	return buf
}
