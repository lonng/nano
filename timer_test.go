package nano

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTimer(t *testing.T) {
	var exists = struct {
		timers        int
		createdTimes  int
		closingTimers int
	}{
		timers:        len(timerManager.timers),
		createdTimes:  len(timerManager.createdTimer),
		closingTimers: len(timerManager.closingTimer),
	}

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

	if len(timerManager.timers) != exists.timers+tc {
		t.Fatalf("timers: %d", len(timerManager.timers))
	}

	if len(timerManager.createdTimer) != exists.createdTimes {
		t.Fatalf("createdTimer: %d", len(timerManager.createdTimer))
	}

	if len(timerManager.closingTimer) != exists.closingTimers {
		t.Fatalf("closingTimer: %d", len(timerManager.closingTimer))
	}
}

func TestNewAfterTimer(t *testing.T) {
	var exists = struct {
		timers        int
		createdTimes  int
		closingTimers int
	}{
		timers:        len(timerManager.timers),
		createdTimes:  len(timerManager.createdTimer),
		closingTimers: len(timerManager.closingTimer),
	}

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

	if len(timerManager.timers) != exists.timers {
		t.Fatalf("timers: %d", len(timerManager.timers))
	}

	if len(timerManager.createdTimer) != exists.createdTimes {
		t.Fatalf("createdTimer: %d", len(timerManager.createdTimer))
	}

	if len(timerManager.closingTimer) != exists.closingTimers {
		t.Fatalf("closingTimer: %d", len(timerManager.closingTimer))
	}
}
