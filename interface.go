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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lonnng/nano/component"
)

// ListenWithOptions start a starx application
func ListenWithOptions(addr string, isWs bool) {
	startupComps()

	go func() {
		if isWs {
			listenAndServeWS(addr)
		} else {
			listenAndServe(addr)
		}
	}()

	log.Println("listen at", addr)
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	// stop server
	select {
	case <-env.die:
		log.Println("The app will shutdown in a few seconds")
	case s := <-sg:
		log.Println("got signal", s)
	}

	log.Println("server is stopping...")

	// shutdown all components registered by application, that
	// call by reverse order against register
	shutdownComps()
}

func Register(c component.Component) {
	comps = append(comps, c)
}

// Set heartbeat time internal
func SetHeartbeatInternal(d time.Duration) {
	env.heartbeat = d
}

// SetCheckOriginFunc set the function that check `Origin` in http headers
func SetCheckOriginFunc(fn func(*http.Request) bool) {
	env.checkOrigin = fn
}

func Shutdown() {
	close(env.die)
}

func EnableDebug() {
	env.debug = true
}

// Callback when session closed
// Waring: session has closed,
func OnSessionClosed(cb SessionClosedHandler) {
	env.muCallbacks.Lock()
	defer env.muCallbacks.Unlock()

	env.callbacks = append(env.callbacks, cb)
}
