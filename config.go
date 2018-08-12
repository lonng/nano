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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jmesyan/nano/session"
)

// VERSION returns current nano version
var VERSION = "0.0.1"

var (
	// app represents the current server process
	app = &struct {
		name    string    // current application name
		startAt time.Time // startup time
	}{}

	// env represents the environment of the current process, includes
	// work path and config path etc.
	env = &struct {
		wd                   string                   // working path
		die                  chan bool                // wait for end application
		heartbeat            time.Duration            // heartbeat internal
		heartbeatTimeout     time.Duration            // heartbeat timeout
		nextHeartbeatTimeout time.Time                // nextHeartbeat timeout time
		checkOrigin          func(*http.Request) bool // check origin when websocket enabled
		debug                bool                     // enable debug
		wsPath               string                   // WebSocket path(eg: ws://127.0.0.1/wsPath)

		// session closed handlers
		muCallbacks sync.RWMutex           // protect callbacks
		callbacks   []SessionClosedHandler // callbacks that emitted on session closed
	}{}

	reconnect = &struct {
		isreconnect          bool
		addr                 string
		opts                 []Option
		reconnectAttempts    int64
		reconnectMaxAttempts int64
		reconnectionDelay    time.Duration
	}{}
)

type (
	// SessionClosedHandler represents a callback that will be called when a session
	// close or session low-level connection broken.
	SessionClosedHandler func(session *session.Session)
)

// init default configs
func init() {
	// application initialize
	app.name = strings.TrimLeft(filepath.Base(os.Args[0]), "/")
	app.startAt = time.Now()

	// environment initialize
	if wd, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		env.wd, _ = filepath.Abs(wd)
	}

	env.die = make(chan bool)
	env.heartbeat = 30 * time.Second
	env.debug = false
	env.muCallbacks = sync.RWMutex{}
	env.checkOrigin = func(_ *http.Request) bool { return true }

	reconnect.isreconnect = true
	reconnect.reconnectAttempts = 0
	reconnect.reconnectMaxAttempts = 10
	reconnect.reconnectionDelay = time.Duration(5) * time.Second
}
