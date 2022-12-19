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

package message

import (
	"encoding/binary"
	"errors"
	"fmt"
	originLog "log"
	"strings"
	"time"

	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/pkg/utils/encoding"
	throwV1 "github.com/suhanyujie/throw_interface/golang_pb/throw/v1"
	"google.golang.org/protobuf/proto"
)

// Type represents the type of message, which could be Request/Notify/Response/Push
type Type byte

// Message types
const (
	Request  Type = 0x00
	Notify        = 0x01
	Response      = 0x02
	Push          = 0x03
)

const (
	msgRouteCompressMask = 0x01
	msgTypeMask          = 0x07
	msgRouteLengthMask   = 0xFF
	msgHeadLength        = 0x02
)

var types = map[Type]string{
	Request:  "Request",
	Notify:   "Notify",
	Response: "Response",
	Push:     "Push",
}

func (t Type) String() string {
	return types[t]
}

var (
	routes = make(map[string]uint16) // route map to code
	codes  = make(map[uint16]string) // code map to route
)

// Errors that could be occurred in message codec
var (
	ErrWrongMessageType  = errors.New("wrong message type")
	ErrInvalidMessage    = errors.New("invalid message")
	ErrRouteInfoNotFound = errors.New("route info not found in dictionary")
	ErrWrongMessage      = errors.New("wrong message")
)

// Message represents a unmarshaled message or a message which to be marshaled
type Message struct {
	Type       Type   // message type
	ID         uint64 // unique id, zero while notify mode
	Route      string // route for locating service
	Action     string // Action、Method 的作用也是类似于 Route
	Method     string
	Data       []byte                    // payload
	DataOfPb   *throwV1.IRequestProtocol // payload, struct obj
	IsCompress bool                      // 和前端商定的字段，为 true 表示 data 是 pb 数据
	compressed bool                      // is message compressed nano 库带的字段，暂时不用
}

// New returns a new message instance
func New() *Message {
	return &Message{}
}

// String, implementation of fmt.Stringer interface
func (m *Message) String() string {
	return fmt.Sprintf("%s %s (%dbytes)", types[m.Type], m.Route, len(m.Data))
}

// Encode marshals message to binary format.
func (m *Message) Encode() ([]byte, error) {
	return Encode(m)
}

func routable(t Type) bool {
	return t == Request || t == Notify || t == Push
}

func invalidType(t Type) bool {
	return t < Request || t > Push

}

func Encode(m *Message) ([]byte, error) {
	var err error
	bArr := make([]byte, 0)
	if invalidType(m.Type) {
		return nil, ErrWrongMessageType
	}
	// 和前端商定，无论如何后端都是发送 IResponseProtocol 数据
	switch m.Type {
	case Response, Request, Push, Notify:
		resp := throwV1.IResponseProtocol{
			Code:       0,
			IsCompress: true,
			Data:       nil,
			Callback:   "LoginAction_loadPlayer",
		}
		if resp.IsCompress {
			bArr, err = proto.Marshal(m.DataOfPb)
			if err != nil {
				return bArr, err
			}
			resp.Data = bArr
		} else {
			jsonBytes, _ := encoding.GetJsonCodec().Marshal(m.DataOfPb)
			resp.Data = jsonBytes
		}
		var respIf interface{}
		respIf = &resp
		if _, isProto := respIf.(proto.Message); isProto {
			originLog.Printf("[Encode] assert is proto")
		} else {
			originLog.Printf("[Encode] assert is not proto")
		}
		respJsonBytes, err := env.Serializer.Marshal(respIf)
		if err != nil {
			originLog.Printf("[err][Encode] env.Serializer.Marshal err: %v", err)
			return nil, nil
		}
		return respJsonBytes, nil
	default:
		originLog.Printf("[Encode] 向客户端发消息时，未知的消息类型 \n")
	}

	return nil, nil
}

// Encode marshals message to binary format. Different message types is corresponding to
// different message header, message types is identified by 2-4 bit of flag field. The
// relationship between message types and message header is presented as follows:
// ------------------------------------------
// |   type   |  flag  |       other        |
// |----------|--------|--------------------|
// | request  |----000-|<message id>|<route>|
// | notify   |----001-|<route>             |
// | response |----010-|<message id>        |
// | push     |----011-|<route>             |
// ------------------------------------------
// The figure above indicates that the bit does not affect the type of message.
// See ref: https://github.com/lonnng/nano/blob/master/docs/communication_protocol.md
func EncodeOld(m *Message) ([]byte, error) {
	if invalidType(m.Type) {
		return nil, ErrWrongMessageType
	}

	buf := make([]byte, 0)
	flag := byte(m.Type) << 1

	code, compressed := routes[m.Route]
	if compressed {
		flag |= msgRouteCompressMask
	}
	buf = append(buf, flag)

	if m.Type == Request || m.Type == Response {
		n := m.ID
		// variant length encode
		for {
			b := byte(n % 128)
			n >>= 7
			if n != 0 {
				buf = append(buf, b+128)
			} else {
				buf = append(buf, b)
				break
			}
		}
	}

	if routable(m.Type) {
		if compressed {
			buf = append(buf, byte((code>>8)&0xFF))
			buf = append(buf, byte(code&0xFF))
		} else {
			buf = append(buf, byte(len(m.Route)))
			buf = append(buf, []byte(m.Route)...)
		}
	}

	buf = append(buf, m.Data...)
	return buf, nil
}

// Decode 将消息转换为 Message
func Decode(data *throwV1.IRequestProtocol) (*Message, error) {
	m := New()
	m.ID = uint64(time.Now().UnixMilli())
	m.Type = Request
	m.Action = data.Action
	m.Method = data.Method
	m.Route = strings.ToLower(fmt.Sprintf("%s.%s", data.Action, data.Method))
	m.Data = data.Data
	m.DataOfPb = data

	return m, nil
}

// Decode unmarshal the bytes slice to a message
// See ref: https://github.com/lonnng/nano/blob/master/docs/communication_protocol.md
func DecodeOld(data []byte) (*Message, error) {
	if len(data) < msgHeadLength {
		return nil, ErrInvalidMessage
	}
	m := New()
	flag := data[0]
	offset := 1
	m.Type = Type((flag >> 1) & msgTypeMask)

	if invalidType(m.Type) {
		return nil, ErrWrongMessageType
	}

	if m.Type == Request || m.Type == Response {
		id := uint64(0)
		// little end byte order
		// WARNING: must can be stored in 64 bits integer
		// variant length encode
		for i := offset; i < len(data); i++ {
			b := data[i]
			id += uint64(b&0x7F) << uint64(7*(i-offset))
			if b < 128 {
				offset = i + 1
				break
			}
		}
		m.ID = id
	}

	if offset >= len(data) {
		return nil, ErrWrongMessage
	}

	if routable(m.Type) {
		if flag&msgRouteCompressMask == 1 {
			m.compressed = true
			code := binary.BigEndian.Uint16(data[offset:(offset + 2)])
			route, ok := codes[code]
			if !ok {
				return nil, ErrRouteInfoNotFound
			}
			m.Route = route
			offset += 2
		} else {
			m.compressed = false
			rl := data[offset]
			offset++
			if offset+int(rl) > len(data) {
				return nil, ErrWrongMessage
			}
			m.Route = string(data[offset:(offset + int(rl))])
			offset += int(rl)
		}
	}

	if offset > len(data) {
		return nil, ErrWrongMessage
	}
	m.Data = data[offset:]
	return m, nil
}

// SetDictionary set routes map which be used to compress route.
// TODO(warning): set dictionary in runtime would be a dangerous operation!!!!!!
func SetDictionary(dict map[string]uint16) {
	for route, code := range dict {
		r := strings.TrimSpace(route)

		// duplication check
		if _, ok := routes[r]; ok {
			log.Println(fmt.Sprintf("duplicated route(route: %s, code: %d)", r, code))
		}

		if _, ok := codes[code]; ok {
			log.Println(fmt.Sprintf("duplicated route(route: %s, code: %d)", r, code))
		}

		// update map, using last value when key duplicated
		routes[r] = code
		codes[code] = r
	}
}

func GetDictionary() (map[string]uint16, bool) {
	if len(routes) <= 0 {
		return nil, false
	}
	return routes, true
}
