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
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package nano

import (
	"errors"
	"sync"

	"github.com/lonnng/nano/codec"
	"github.com/lonnng/nano/packet"
	"github.com/lonnng/nano/session"
)

var (
	heartbeatPacket, _ = codec.Encode(packet.Heartbeat, nil)
	// transporter represents a manager, which manages low-level transport
	// layer object, that abstract as `agent` in frontend server or `acceptor`
	// in the backend server
	transporter        = newTransporter()
	ErrSessionOnNotify = errors.New("current session working on notify mode")
)

type transportService struct {
	sync.RWMutex
	sessionCloseCb []func(*session.Session) // callback on session closed
}

// Create new t service
func newTransporter() *transportService {
	return &transportService{}
}

func (t *transportService) sessionClosedCallback(cb func(*session.Session)) {
	t.Lock()
	defer t.Unlock()

	t.sessionCloseCb = append(t.sessionCloseCb, cb)
}

// Callback when session closed
// Waring: session has closed,
func OnSessionClosed(cb func(*session.Session)) {
	transporter.sessionClosedCallback(cb)
}
