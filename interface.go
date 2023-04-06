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

package nano

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/internal/runtime"
	"github.com/lonng/nano/scheduler"
)

var running int32

// VERSION returns current nano version
var VERSION = "0.5.0"

var (
	// app represents the current server process
	app = &struct {
		name    string    // current application name
		startAt time.Time // startup time
	}{}
)

// Listen listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.
func Listen(addr string, opts ...Option) {
	if atomic.AddInt32(&running, 1) != 1 {
		log.Println("Nano has running")
		return
	}

	// application initialize
	app.name = strings.TrimLeft(filepath.Base(os.Args[0]), "/")
	app.startAt = time.Now()

	// environment initialize
	if wd, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		env.Wd, _ = filepath.Abs(wd)
	}

	opt := cluster.Options{
		Components: &component.Components{},
	}
	for _, option := range opts {
		option(&opt)
	}

	// Use listen address as client address in non-cluster mode
	if !opt.IsMaster && opt.AdvertiseAddr == "" && opt.ClientAddr == "" {
		log.Println("The current server running in singleton mode")
		opt.ClientAddr = addr
	}

	// Set the retry interval to 3 secondes if doesn't set by user
	if opt.RetryInterval == 0 {
		opt.RetryInterval = time.Second * 3
	}

	node := &cluster.Node{
		Options:     opt,
		ServiceAddr: addr,
	}
	err := node.Startup()
	if err != nil {
		log.Fatalf("Node startup failed: %v", err)
	}
	runtime.CurrentNode = node

	if node.ClientAddr != "" {
		log.Println(fmt.Sprintf("Startup *Nano gate server* %s, client address: %v, service address: %s",
			app.name, node.ClientAddr, node.ServiceAddr))
	} else {
		log.Println(fmt.Sprintf("Startup *Nano backend server* %s, service address %s",
			app.name, node.ServiceAddr))
	}

	go scheduler.Sched()
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	select {
	case <-env.Die:
		log.Println("The app will shutdown in a few seconds")
	case s := <-sg:
		log.Println("Nano server got signal", s)
		node.Options.SignalHandler()
	}

	log.Println("Nano server is stopping...")

	node.Shutdown()
	runtime.CurrentNode = nil
	scheduler.Close()
	atomic.StoreInt32(&running, 0)
}

// Shutdown send a signal to let 'nano' shutdown itself.
func Shutdown() {
	close(env.Die)
}
