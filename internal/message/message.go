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
// THE SOFTWARE IS PROVMidED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package message

import (
	// "encoding/binary"
	"errors"
	// "fmt"
	"log"
	"strings"
)

// int32 represents the type of message, which could be Request/Notify/Response/Push

// Message types
const (
	Request  int32 = 0x00
	Notify         = 0x01
	Response       = 0x02
	Push           = 0x03
)

const (
	msgRouteCompressMask = 0x01
	msgTypeMask          = 0x07
	msgRouteLengthMask   = 0xFF
	msgHeadLength        = 0x02
)

var types = map[int32]string{
	Request:  "Request",
	Notify:   "Notify",
	Response: "Response",
	Push:     "Push",
}

var (
	routes = make(map[string]uint16) // route map to code
	codes  = make(map[uint16]string) // code map to route
)

// Errors that could be occurred in message codec
var (
	ErrWrongMessageType  = errors.New("wrong message type")
	ErrInvalMidMessage   = errors.New("invalMid message")
	ErrRouteInfoNotFound = errors.New("route info not found in dictionary")
)

// // Message represents a unmarshaled message or a message which to be marshaled
// type Message struct {
// 	Type  int32  // message type
// 	Mid   int32  // unique Mid, zero while notify mode
// 	Route string // route for locating service
// 	Data  []byte // payload
// 	// compressed bool   // is message compressed
// }

// // New returns a new message instance
// func New() *Message {
// 	return &Message{}
// }

// // String, implementation of fmt.Stringer interface
// func (m *Message) String() string {
// 	return fmt.Sprintf("int32: %s, Mid: %d, Route: %s,  BodyLength: %d",
// 		m.Type,
// 		m.Mid,
// 		m.Route,
// 		len(m.Data))
// }

// // Encode marshals message to binary format.
// func (m *Message) Encode() ([]byte, error) {
// 	return Encode(m)
// }

// func routable(t int32) bool {
// 	return t == Request || t == Notify || t == Push
// }

// func invalMidType(t int32) bool {
// 	return t < Request || t > Push

// }

// // Encode marshals message to binary format. Different message types is corresponding to
// // different message header, message types is Midentified by 2-4 bit of flag field. The
// // relationship between message types and message header is presented as follows:
// // ------------------------------------------
// // |   type   |  flag  |       other        |
// // |----------|--------|--------------------|
// // | request  |----000-|<message Mid>|<route>|
// // | notify   |----001-|<route>             |
// // | response |----010-|<message Mid>        |
// // | push     |----011-|<route>             |
// // ------------------------------------------
// // The figure above indicates that the bit does not affect the type of message.
// // See ref: https://github.com/jmesyan/nano/blob/master/docs/communication_protocol.md
// func Encode(m *Message) ([]byte, error) {
// 	if invalMidType(m.Type) {
// 		return nil, ErrWrongMessageType
// 	}

// 	buf := make([]byte, 0)
// 	flag := byte(m.Type) << 1

// 	code, compressed := routes[m.Route]
// 	if compressed {
// 		flag |= msgRouteCompressMask
// 	}
// 	buf = append(buf, flag)

// 	if m.Type == Request || m.Type == Response {
// 		n := m.Mid
// 		// variant length encode
// 		for {
// 			b := byte(n % 128)
// 			n >>= 7
// 			if n != 0 {
// 				buf = append(buf, b+128)
// 			} else {
// 				buf = append(buf, b)
// 				break
// 			}
// 		}
// 	}

// 	if routable(m.Type) {
// 		if compressed {
// 			buf = append(buf, byte((code>>8)&0xFF))
// 			buf = append(buf, byte(code&0xFF))
// 		} else {
// 			buf = append(buf, byte(len(m.Route)))
// 			buf = append(buf, []byte(m.Route)...)
// 		}
// 	}

// 	buf = append(buf, m.Data...)
// 	return buf, nil
// }

// // Decode unmarshal the bytes slice to a message
// // See ref: https://github.com/jmesyan/nano/blob/master/docs/communication_protocol.md
// func Decode(data []byte) (*Message, error) {
// 	if len(data) < msgHeadLength {
// 		return nil, ErrInvalMidMessage
// 	}
// 	m := New()
// 	flag := data[0]
// 	offset := 1
// 	m.Type = int32((flag >> 1) & msgTypeMask)

// 	if invalMidType(m.Type) {
// 		return nil, ErrWrongMessageType
// 	}

// 	if m.Type == Request || m.Type == Response {
// 		Mid := int32(0)
// 		// little end byte order
// 		// WARNING: must can be stored in 64 bits integer
// 		// variant length encode
// 		for i := offset; i < len(data); i++ {
// 			b := data[i]
// 			Mid += int32(b&0x7F) << int32(7*(i-offset))
// 			if b < 128 {
// 				offset = i + 1
// 				break
// 			}
// 		}
// 		m.Mid = Mid
// 	}

// 	if routable(m.Type) {
// 		if flag&msgRouteCompressMask == 1 {
// 			m.compressed = true
// 			code := binary.BigEndian.Uint16(data[offset:(offset + 2)])
// 			route, ok := codes[code]
// 			if !ok {
// 				return nil, ErrRouteInfoNotFound
// 			}
// 			m.Route = route
// 			offset += 2
// 		} else {
// 			m.compressed = false
// 			rl := data[offset]
// 			offset++
// 			m.Route = string(data[offset:(offset + int(rl))])
// 			offset += int(rl)
// 		}
// 	}

// 	m.Data = data[offset:]
// 	return m, nil
// }

// SetDictionary set routes map which be used to compress route.
// TODO(warning): set dictionary in runtime would be a dangerous operation!!!!!!
func SetDictionary(dict map[string]uint16) {
	for route, code := range dict {
		r := strings.TrimSpace(route)

		// duplication check
		if _, ok := routes[r]; ok {
			log.Printf("duplicated route(route: %s, code: %d)\n", r, code)
		}

		if _, ok := codes[code]; ok {
			log.Printf("duplicated route(route: %s, code: %d)\n", r, code)
		}

		// update map, using last value when key duplicated
		routes[r] = code
		codes[code] = r
	}
}
