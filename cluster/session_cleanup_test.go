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
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/session"
)

func TestLocalSessionRemovedOnClientDisconnect(t *testing.T) {
	node := &Node{
		sessions: map[int64]*session.Session{},
	}
	h := NewHandler(node, nil)

	clientConn, serverConn := net.Pipe()

	done := make(chan struct{})
	go func() {
		h.handle(serverConn)
		close(done)
	}()

	var sid int64
	deadline := time.Now().Add(2 * time.Second)
	for {
		node.mu.RLock()
		count := len(node.sessions)
		if count == 1 {
			for id := range node.sessions {
				sid = id
			}
			node.mu.RUnlock()
			break
		}
		node.mu.RUnlock()
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for session to be stored in node")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if s := node.findSession(sid); s == nil {
		t.Fatalf("expected session %d to be found, got nil", sid)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("failed to close client connection: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for handle goroutine to exit")
	}

	node.mu.RLock()
	remaining := len(node.sessions)
	node.mu.RUnlock()
	if remaining != 0 {
		t.Fatalf("expected 0 sessions in node after disconnect, found %d", remaining)
	}

	if s := node.findSession(sid); s != nil {
		t.Fatalf("expected session %d to be removed from node, but still found", sid)
	}
}

func TestMultipleClientsConcurrentDisconnect(t *testing.T) {
	node := &Node{
		sessions: map[int64]*session.Session{},
	}
	h := NewHandler(node, nil)

	const numClients = 10
	clientConns := make([]net.Conn, numClients)
	serverConns := make([]net.Conn, numClients)
	dones := make([]chan struct{}, numClients)

	for i := 0; i < numClients; i++ {
		clientConns[i], serverConns[i] = net.Pipe()
		dones[i] = make(chan struct{})
		go func(idx int) {
			h.handle(serverConns[idx])
			close(dones[idx])
		}(i)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		node.mu.RLock()
		count := len(node.sessions)
		node.mu.RUnlock()
		if count == numClients {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d sessions, got %d", numClients, count)
		}
		time.Sleep(10 * time.Millisecond)
	}

	var wg sync.WaitGroup
	for i := 0; i < numClients/2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientConns[idx].Close()
			<-dones[idx]
		}(i)
	}
	wg.Wait()

	node.mu.RLock()
	halfCount := len(node.sessions)
	node.mu.RUnlock()
	if halfCount != numClients/2 {
		t.Fatalf("expected %d sessions remaining, got %d", numClients/2, halfCount)
	}

	for i := numClients / 2; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			clientConns[idx].Close()
			<-dones[idx]
		}(i)
	}
	wg.Wait()

	node.mu.RLock()
	finalCount := len(node.sessions)
	node.mu.RUnlock()
	if finalCount != 0 {
		t.Fatalf("expected 0 sessions remaining, got %d", finalCount)
	}
}

func TestLocalSessionRemovedOnServerSideClose(t *testing.T) {
	node := &Node{
		sessions: map[int64]*session.Session{},
	}
	h := NewHandler(node, nil)

	clientConn, serverConn := net.Pipe()

	done := make(chan struct{})
	go func() {
		h.handle(serverConn)
		close(done)
	}()

	var sid int64
	var s *session.Session
	deadline := time.Now().Add(2 * time.Second)
	for {
		node.mu.RLock()
		if len(node.sessions) == 1 {
			for id, sess := range node.sessions {
				sid = id
				s = sess
			}
			node.mu.RUnlock()
			break
		}
		node.mu.RUnlock()
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for session store")
		}
		time.Sleep(10 * time.Millisecond)
	}

	s.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for handle to exit after server close")
	}

	if sess := node.findSession(sid); sess != nil {
		t.Fatalf("expected session %d to be removed, but still found", sid)
	}

	clientConn.Close()
}

func TestCloseSessionRPC(t *testing.T) {
	node := &Node{
		sessions: map[int64]*session.Session{},
	}
	h := NewHandler(node, nil)

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		h.handle(serverConn)
		close(done)
	}()

	var sid int64
	deadline := time.Now().Add(2 * time.Second)
	for {
		node.mu.RLock()
		if len(node.sessions) == 1 {
			for id := range node.sessions {
				sid = id
			}
			node.mu.RUnlock()
			break
		}
		node.mu.RUnlock()
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for session store")
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, err := node.CloseSession(context.Background(), &clusterpb.CloseSessionRequest{SessionId: sid})
	if err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for handle to exit after CloseSession RPC")
	}

	if sess := node.findSession(sid); sess != nil {
		t.Fatalf("expected session %d to be deleted, but still found", sid)
	}
}

func TestDeleteSessionIdempotency(t *testing.T) {
	node := &Node{
		sessions: map[int64]*session.Session{},
	}

	node.deleteSession(12345)

	s := session.New(nil)
	node.storeSession(s)
	if node.findSession(s.ID()) == nil {
		t.Fatalf("expected session %d to exist", s.ID())
	}

	node.deleteSession(s.ID())
	if node.findSession(s.ID()) != nil {
		t.Fatalf("expected session %d to be deleted", s.ID())
	}

	node.deleteSession(s.ID())
	if node.findSession(s.ID()) != nil {
		t.Fatalf("expected session %d to remain deleted", s.ID())
	}
}

func TestSessionClosedRPC(t *testing.T) {
	node := &Node{
		sessions: map[int64]*session.Session{},
	}

	s := session.New(nil)
	node.storeSession(s)

	_, err := node.SessionClosed(context.Background(), &clusterpb.SessionClosedRequest{SessionId: s.ID()})
	if err != nil {
		t.Fatalf("SessionClosed failed: %v", err)
	}

	if node.findSession(s.ID()) != nil {
		t.Fatalf("expected session %d to be deleted by SessionClosed", s.ID())
	}
}

func TestSessionLifetimeOnClosedFiredOnce(t *testing.T) {
	node := &Node{
		sessions: map[int64]*session.Session{},
	}
	h := NewHandler(node, nil)

	var closedCount int64
	var targetSid int64

	session.Lifetime.OnClosed(func(s *session.Session) {
		if s.ID() == targetSid {
			atomic.AddInt64(&closedCount, 1)
		}
	})

	clientConn, serverConn := net.Pipe()

	done := make(chan struct{})
	go func() {
		h.handle(serverConn)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		node.mu.RLock()
		if len(node.sessions) == 1 {
			for id := range node.sessions {
				targetSid = id
			}
			node.mu.RUnlock()
			break
		}
		node.mu.RUnlock()
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for session store")
		}
		time.Sleep(10 * time.Millisecond)
	}

	clientConn.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for handle exit")
	}

	if node.findSession(targetSid) != nil {
		t.Fatalf("expected session %d to be removed", targetSid)
	}
}
