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

package cluster

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lonng/nano/internal/env"
	"google.golang.org/grpc"
)

type connPool struct {
	index uint32
	v     []*grpc.ClientConn
}

type rpcClient struct {
	sync.RWMutex
	isClosed bool
	pools    map[string]*connPool
}

func newConnArray(maxSize uint, addr string) (*connPool, error) {
	a := &connPool{
		index: 0,
		v:     make([]*grpc.ClientConn, maxSize),
	}
	if err := a.init(addr); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *connPool) init(addr string) error {
	for i := range a.v {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		conn, err := grpc.DialContext(
			ctx,
			addr,
			env.GrpcOptions...,
		)
		cancel()
		if err != nil {
			// Cleanup if the initialization fails.
			a.Close()
			return err
		}
		a.v[i] = conn

	}
	return nil
}

func (a *connPool) Get() *grpc.ClientConn {
	next := atomic.AddUint32(&a.index, 1) % uint32(len(a.v))
	return a.v[next]
}

func (a *connPool) Close() {
	for i, c := range a.v {
		if c != nil {
			err := c.Close()
			if err != nil {
				// TODO: error handling
			}
			a.v[i] = nil
		}
	}
}

func newRPCClient() *rpcClient {
	return &rpcClient{
		pools: make(map[string]*connPool),
	}
}

func (c *rpcClient) getConnPool(addr string) (*connPool, error) {
	c.RLock()
	if c.isClosed {
		c.RUnlock()
		return nil, errors.New("rpc client is closed")
	}
	array, ok := c.pools[addr]
	c.RUnlock()
	if !ok {
		var err error
		array, err = c.createConnPool(addr)
		if err != nil {
			return nil, err
		}
	}
	return array, nil
}

func (c *rpcClient) createConnPool(addr string) (*connPool, error) {
	c.Lock()
	defer c.Unlock()
	array, ok := c.pools[addr]
	if !ok {
		var err error
		// TODO: make conn count configurable
		array, err = newConnArray(10, addr)
		if err != nil {
			return nil, err
		}
		c.pools[addr] = array
	}
	return array, nil
}

func (c *rpcClient) closePool() {
	c.Lock()
	if !c.isClosed {
		c.isClosed = true
		// close all connections
		for _, array := range c.pools {
			array.Close()
		}
	}
	c.Unlock()
}
