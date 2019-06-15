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

package scheduler

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
