package io

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/lonnng/nano"
	"github.com/lonnng/nano/benchmark/testdata"
	"github.com/lonnng/nano/component"
	"github.com/lonnng/nano/serialize/protobuf"
	"github.com/lonnng/nano/session"
	"log"
)

const (
	addr = "127.0.0.1:3250" // local address
	conc = 100              // concurrent client count
)

type TestHandler struct {
	component.Base
	metrics int32
}

func (h *TestHandler) AfterInit() {
	ticker := time.NewTicker(time.Second)

	// metrics output ticker
	go func() {
		for range ticker.C {
			println("QPS", atomic.LoadInt32(&h.metrics))
			atomic.StoreInt32(&h.metrics, 0)
		}
	}()
}

func (h *TestHandler) Ping(s *session.Session, data *testdata.Ping) error {
	atomic.AddInt32(&h.metrics, 1)
	return s.Response(&testdata.Pong{Content: data.Content})
}

func server() {
	nano.Register(&TestHandler{})
	nano.SetSerializer(protobuf.NewSerializer())

	nano.ListenWithOptions(addr, false)
}

func client() {
	c := NewConnector()

	if err := c.Start(addr); err != nil {
		panic(err)
	}

	c.OnConnected(func() {
		var i = 0
		for i < 10 {
			i++
			c.Request("TestHandler.Ping", &testdata.Ping{}, func(data interface{}) {
				println("pong")
			})
		}
	})
}

func TestIO(t *testing.T) {
	go server()
	go client()

	log.SetFlags(log.LstdFlags | log.Llongfile)

	sg := make(chan os.Signal)
	signal.Notify(sg, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	<-sg

	t.Log("exit")
}
