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
	"sync/atomic"
	"syscall"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/internal/runtime"
	"github.com/lonng/nano/scheduler"
)

var running int32

func run(addr string, isWs bool, certificate string, key string, opts ...Option) {
	if atomic.AddInt32(&running, 1) != 1 {
		log.Println("Nano has running")
		return
	}

	opt := &options{
		components: &component.Components{},
	}
	for _, option := range opts {
		option(opt)
	}

	node := &cluster.Node{
		Label:          opt.label,
		IsMaster:       opt.isMaster,
		AdvertiseAddr:  opt.advertiseAddr,
		MemberAddr:     opt.memberAddr,
		ServerAddr:     addr,
		Components:     opt.components,
		IsWebsocket:    isWs,
		TSLCertificate: certificate,
		TSLKey:         key,
		Pipeline:       opt.pipeline,
	}
	err := node.Startup()
	if err != nil {
		log.Fatalf("Node startup failed: %v", err)
	}
	runtime.CurrentNode = node

	log.Println(fmt.Sprintf("Nano server %s started, listen at %s", app.name, addr))
	scheduler.Sched()
	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	select {
	case <-env.Die:
		log.Println("The app will shutdown in a few seconds")
	case s := <-sg:
		log.Println("Nano server got signal", s)
	}

	log.Println("Nano server is stopping...")

	node.Shutdown()
	runtime.CurrentNode = nil
	scheduler.Close()
	atomic.StoreInt32(&running, 0)
}
