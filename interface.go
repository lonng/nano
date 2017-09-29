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
	"net/http"
	"time"

	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/internal/message"
)

// Listen listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.
func Listen(addr string) {
	listen(addr, false)
}

// ListenWS listens on the TCP network address addr
// and then upgrades the HTTP server connection to the WebSocket protocol
// to handle requests on incoming connections.
func ListenWS(addr string) {
	listen(addr, true)
}

// Register register a component with options
func Register(c component.Component, options ...component.Option) {
	comps = append(comps, regComp{c, options})
}

// SetHeartbeatInterval set heartbeat time interval
func SetHeartbeatInterval(d time.Duration) {
	env.heartbeat = d
}

// SetCheckOriginFunc set the function that check `Origin` in http headers
func SetCheckOriginFunc(fn func(*http.Request) bool) {
	env.checkOrigin = fn
}

// Shutdown send a signal to let 'nano' shutdown itself.
func Shutdown() {
	close(env.die)
}

// EnableDebug let 'nano' to run under debug mode.
func EnableDebug() {
	env.debug = true
}

// OnSessionClosed set the Callback which will be called when session is closed
// Waring: session has closed,
func OnSessionClosed(cb SessionClosedHandler) {
	env.muCallbacks.Lock()
	defer env.muCallbacks.Unlock()

	env.callbacks = append(env.callbacks, cb)
}

// SetDictionary set routes map, TODO(warning): set dictionary in runtime would be a dangerous operation!!!!!!
func SetDictionary(dict map[string]uint16) {
	message.SetDictionary(dict)
}
