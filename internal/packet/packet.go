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

package packet

import (
	"errors"
	"fmt"
)

// Type represents the network packet's type such as: handshake and so on.
type Type byte

const (
	_ Type = iota
	// Handshake represents a handshake: request(client) <====> handshake response(server)
	Handshake = 0x01

	// HandshakeAck represents a handshake ack from client to server
	HandshakeAck = 0x02

	// Heartbeat represents a heartbeat
	Heartbeat = 0x03

	// Data represents a common data packet
	Data = 0x04

	// Kick represents a kick off packet
	Kick = 0x05 // disconnect message from server
)

// ErrWrongPacketType represents a wrong packet type.
var ErrWrongPacketType = errors.New("wrong packet type")

// Packet represents a network packet.
type Packet struct {
	Type   Type
	Length int
	Data   []byte
}

//New create a Packet instance.
func New() *Packet {
	return &Packet{}
}

//String represents the Packet's in text mode.
func (p *Packet) String() string {
	return fmt.Sprintf("Type: %d, Length: %d, Data: %s", p.Type, p.Length, string(p.Data))
}
