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
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
)

// Listen listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.
func Listen(addr string, opts ...Option) {
	run(addr, false, "", "", opts...)
}

// ListenWS listens on the TCP network address addr
// and then upgrades the HTTP server connection to the WebSocket protocol
// to handle requests on incoming connections.
func ListenWS(addr string, opts ...Option) {
	run(addr, true, "", "", opts...)
}

// ListenWS listens on the TCP network address addr
// and then upgrades the HTTP server connection to the WebSocket protocol
// to handle requests on incoming connections.
func ListenWSTLS(addr string, certificate string, key string, opts ...Option) {
	run(addr, true, certificate, key, opts...)
}

// Shutdown send a signal to let 'nano' shutdown itself.
func Shutdown() {
	close(env.Die)
}

// SetLogger rewrites the default logger
func SetLogger(l log.Logger) {
	log.SetLogger(l)
}
