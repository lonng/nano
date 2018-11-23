package nano

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTimer(t *testing.T) {
	const tc = 1000
	var counter int64
	for i := 0; i < tc; i++ {
		NewTimer(1*time.Millisecond, func() {
			atomic.AddInt64(&counter, 1)
		})
	}

	<-time.After(5 * time.Millisecond)
	cron()
	cron()
	if counter != tc*2 {
		t.Fatalf("expect: %d, got: %d", tc*2, counter)
	}

	if len(timerManager.timers) != tc {
		t.Fatalf("timers: %d", len(timerManager.timers))
	}

	if len(timerManager.createdTimer) != 0 {
		t.Fatalf("createdTimer: %d", len(timerManager.createdTimer))
	}

	if len(timerManager.closingTimer) != 0 {
		t.Fatalf("closingTimer: %d", len(timerManager.closingTimer))
	}
}

func TestNewAfterTimer(t *testing.T) {
	const tc = 1000
	var counter int64
	for i := 0; i < tc; i++ {
		NewAfterTimer(1*time.Millisecond, func() {
			atomic.AddInt64(&counter, 1)
		})
	}

	<-time.After(5 * time.Millisecond)
	cron()
	if counter != tc {
		t.Fatalf("expect: %d, got: %d", tc, counter)
	}

	if len(timerManager.timers) != 0 {
		t.Fatalf("timers: %d", len(timerManager.timers))
	}

	if len(timerManager.createdTimer) != 0 {
		t.Fatalf("createdTimer: %d", len(timerManager.createdTimer))
	}

	if len(timerManager.closingTimer) != 0 {
		t.Fatalf("closingTimer: %d", len(timerManager.closingTimer))
	}
}
