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

package mock

import (
	"fmt"
	"net"
)

// NetAddr mock the net.Addr interface
type NetAddr struct{}

// Network implements the net.Addr interface
func (a NetAddr) Network() string { return "mock" }

// String implements the net.Addr interface
func (a NetAddr) String() string { return "mock-addr" }

type message struct {
	route string
	data  interface{}
}

// NetworkEntity represents an network entity which can be used to construct the
// session object.
type NetworkEntity struct {
	messages  []message
	responses []interface{}
	msgmap    map[uint64]interface{}
	rpcCall   []message
}

// NewNetworkEntity returns an mock network entity
func NewNetworkEntity() *NetworkEntity {
	return &NetworkEntity{
		msgmap: map[uint64]interface{}{},
	}
}

// RPC implements the session.NetworkEntity interface
func (n *NetworkEntity) RPC(route string, v interface{}) error {
	n.rpcCall = append(n.rpcCall, message{route: route, data: v})
	return nil
}

// Push implements the session.NetworkEntity interface
func (n *NetworkEntity) Push(route string, v interface{}) error {
	n.messages = append(n.messages, message{route: route, data: v})
	return nil
}

// LastMid implements the session.NetworkEntity interface
func (n *NetworkEntity) LastMid() uint64 {
	return 1
}

// Response implements the session.NetworkEntity interface
func (n *NetworkEntity) Response(v interface{}) error {
	n.responses = append(n.responses, v)
	return nil
}

// ResponseMid implements the session.NetworkEntity interface
func (n *NetworkEntity) ResponseMid(mid uint64, v interface{}) error {
	_, found := n.msgmap[mid]
	if found {
		return fmt.Errorf("duplicated message id: %v", mid)
	}
	n.msgmap[mid] = v
	return nil
}

// Close implements the session.NetworkEntity interface
func (n *NetworkEntity) Close() error {
	return nil
}

// RemoteAddr implements the session.NetworkEntity interface
func (n *NetworkEntity) RemoteAddr() net.Addr {
	return NetAddr{}
}

// LastResponse returns the last respond message
func (n *NetworkEntity) LastResponse() interface{} {
	if len(n.responses) < 1 {
		return nil
	}
	return n.responses[len(n.responses)-1]
}

// FindResponseByMID returns the response respective the message id
func (n *NetworkEntity) FindResponseByMID(mid uint64) interface{} {
	return n.msgmap[mid]
}

// FindResponseByRoute returns the response respective the route
func (n *NetworkEntity) FindResponseByRoute(route string) interface{} {
	for i := range n.messages {
		if n.messages[i].route == route {
			return n.messages[i].data
		}
	}
	return nil
}
