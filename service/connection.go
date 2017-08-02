package service

import (
	"sync/atomic"
)

var Connections = newConnectionService()

type connectionService struct {
	count int64
	sid   int64
}

func newConnectionService() *connectionService {
	return &connectionService{sid: 0}
}

func (c *connectionService) Increment() {
	atomic.AddInt64(&c.count, 1)
}

func (c *connectionService) Decrement() {
	atomic.AddInt64(&c.count, -1)
}

func (c *connectionService) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

func (c *connectionService) Reset() {
	atomic.StoreInt64(&c.count, 0)
	atomic.StoreInt64(&c.sid, 0)
}

func (c *connectionService) SessionID() int64 {
	return atomic.AddInt64(&c.sid, 1)
}
