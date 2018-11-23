package nano

import (
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const (
	loopForever = -1
)

var (
	// timerManager manager for all timers
	timerManager = &struct {
		incrementID int64            // auto increment id
		timers      map[int64]*Timer // all timers

		muClosingTimer sync.RWMutex
		closingTimer   []int64
		muCreatedTimer sync.RWMutex
		createdTimer   []*Timer
	}{}

	// timerPrecision indicates the precision of timer, default is time.Second
	timerPrecision = time.Second

	// globalTicker represents global ticker that all cron job will be executed
	// in globalTicker.
	globalTicker *time.Ticker
)

type (
	// TimerFunc represents a function which will be called periodically in main
	// logic gorontine.
	TimerFunc func()

	// TimerCondition represents a checker that returns true when cron job needs
	// to execute
	TimerCondition interface {
		Check(now time.Time) bool
	}

	// Timer represents a cron job
	Timer struct {
		id        int64          // timer id
		fn        TimerFunc      // function that execute
		createAt  int64          // timer create time
		interval  time.Duration  // execution interval
		condition TimerCondition // condition to cron job execution
		elapse    int64          // total elapse time
		closed    int32          // is timer closed
		counter   int            // counter
	}
)

func init() {
	timerManager.timers = map[int64]*Timer{}
}

// ID returns id of current timer
func (t *Timer) ID() int64 {
	return t.id
}

// Stop turns off a timer. After Stop, fn will not be called forever
func (t *Timer) Stop() {
	if atomic.AddInt32(&t.closed, 1) != 1 {
		return
	}

	t.counter = 0
}

// execute job function with protection
func safecall(id int64, fn TimerFunc) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(fmt.Sprintf("Call timer function error, TimerID=%d, Error=%v", id, err))
			println(stack())
		}
	}()

	fn()
}

func cron() {
	if len(timerManager.createdTimer) > 0 {
		timerManager.muCreatedTimer.Lock()
		for _, t := range timerManager.createdTimer {
			timerManager.timers[t.id] = t
		}
		timerManager.createdTimer = timerManager.createdTimer[:0]
		timerManager.muCreatedTimer.Unlock()
	}

	if len(timerManager.timers) < 1 {
		return
	}

	now := time.Now()
	unn := now.UnixNano()
	for id, t := range timerManager.timers {
		if t.counter == loopForever || t.counter > 0 {
			// condition timer
			if t.condition != nil {
				if t.condition.Check(now) {
					safecall(id, t.fn)
				}
				continue
			}

			// execute job
			if t.createAt+t.elapse <= unn {
				safecall(id, t.fn)
				t.elapse += int64(t.interval)

				// update timer counter
				if t.counter != loopForever && t.counter > 0 {
					t.counter--
				}
			}
		}

		if t.counter == 0 {
			timerManager.muClosingTimer.Lock()
			timerManager.closingTimer = append(timerManager.closingTimer, t.id)
			timerManager.muClosingTimer.Unlock()
			continue
		}

	}

	if len(timerManager.closingTimer) > 0 {
		timerManager.muClosingTimer.Lock()
		for _, id := range timerManager.closingTimer {
			delete(timerManager.timers, id)
		}
		timerManager.closingTimer = timerManager.closingTimer[:0]
		timerManager.muClosingTimer.Unlock()
	}
}

// NewTimer returns a new Timer containing a function that will be called
// with a period specified by the duration argument. It adjusts the intervals
// for slow receivers.
// The duration d must be greater than zero; if not, NewTimer will panic.
// Stop the timer to release associated resources.
func NewTimer(interval time.Duration, fn TimerFunc) *Timer {
	return NewCountTimer(interval, loopForever, fn)
}

// NewCountTimer returns a new Timer containing a function that will be called
// with a period specified by the duration argument. After count times, timer
// will be stopped automatically, It adjusts the intervals for slow receivers.
// The duration d must be greater than zero; if not, NewCountTimer will panic.
// Stop the timer to release associated resources.
func NewCountTimer(interval time.Duration, count int, fn TimerFunc) *Timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if interval <= 0 {
		panic("non-positive interval for NewTimer")
	}

	t := &Timer{
		id:       atomic.AddInt64(&timerManager.incrementID, 1),
		fn:       fn,
		createAt: time.Now().UnixNano(),
		interval: interval,
		elapse:   int64(interval), // first execution will be after interval
		counter:  count,
	}

	timerManager.muCreatedTimer.Lock()
	timerManager.createdTimer = append(timerManager.createdTimer, t)
	timerManager.muCreatedTimer.Unlock()
	return t
}

// NewAfterTimer returns a new Timer containing a function that will be called
// after duration that specified by the duration argument.
// The duration d must be greater than zero; if not, NewAfterTimer will panic.
// Stop the timer to release associated resources.
func NewAfterTimer(duration time.Duration, fn TimerFunc) *Timer {
	return NewCountTimer(duration, 1, fn)
}

// NewCondTimer returns a new Timer containing a function that will be called
// when condition satisfied that specified by the condition argument.
// The duration d must be greater than zero; if not, NewCondTimer will panic.
// Stop the timer to release associated resources.
func NewCondTimer(condition TimerCondition, fn TimerFunc) *Timer {
	if condition == nil {
		panic("nano/timer: nil condition")
	}

	t := NewCountTimer(time.Duration(math.MaxInt64), loopForever, fn)
	t.condition = condition

	return t
}

// SetTimerPrecision set the ticker precision, and time precision can not less
// than a Millisecond, and can not change after application running. The default
// precision is time.Second
func SetTimerPrecision(precision time.Duration) {
	if precision < time.Millisecond {
		panic("time precision can not less than a Millisecond")
	}
	timerPrecision = precision
}
