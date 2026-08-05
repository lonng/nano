package cluster

import (
	"net"
	"testing"
	"time"

	"github.com/lonng/nano/session"
)

func TestLocalHandlerDeletesSessionOnDisconnect(t *testing.T) {
	node := &Node{sessions: make(map[int64]*session.Session)}
	node.cluster = newCluster(node)
	handler := NewHandler(node, nil)
	node.handler = handler

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		handler.handle(serverConn)
		close(done)
	}()

	waitForSessionCount(t, node, 1)
	if err := clientConn.Close(); err != nil {
		t.Fatalf("close client connection: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		serverConn.Close()
		t.Fatal("timed out waiting for local handler to exit")
	}

	node.mu.RLock()
	count := len(node.sessions)
	node.mu.RUnlock()
	if count != 0 {
		t.Fatalf("session count after disconnect = %d, want 0", count)
	}
}

func waitForSessionCount(t *testing.T, node *Node, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		node.mu.RLock()
		count := len(node.sessions)
		node.mu.RUnlock()
		if count == want {
			return
		}
		time.Sleep(time.Millisecond)
	}

	node.mu.RLock()
	count := len(node.sessions)
	node.mu.RUnlock()
	t.Fatalf("session count = %d, want %d", count, want)
}
