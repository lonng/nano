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

package service

import (
	"sync/atomic"
)

// Connections is a global variable which is used by session.
var Connections = newConnectionService()

type connectionService struct {
	count int64
	sid   int64
}

func newConnectionService() *connectionService {
	return &connectionService{sid: 0}
}

// Increment increment the connection count
func (c *connectionService) Increment() {
	atomic.AddInt64(&c.count, 1)
}

// Decrement decrement the connection count
func (c *connectionService) Decrement() {
	atomic.AddInt64(&c.count, -1)
}

// Count returns the connection numbers in current
func (c *connectionService) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

// Reset reset the connection service status
func (c *connectionService) Reset() {
	atomic.StoreInt64(&c.count, 0)
	atomic.StoreInt64(&c.sid, 0)
}

// SessionID returns the session id
func (c *connectionService) SessionID() int64 {
	return atomic.AddInt64(&c.sid, 1)
}
