package nano

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

const (
	timerBacklog = 512
	loopForever  = -1
)

var (
	// timerManager manager for all timers
	timerManager = &struct {
		incrementId    int64            // auto increment id
		timers         map[int64]*Timer // all timers
		chClosingTimer chan int64       // timer for closing
		chCreatedTimer chan *Timer
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

	// Timer represents a cron job
	Timer struct {
		id       int64         // timer id
		fn       TimerFunc     // function that execute
		createAt int64         // timer create time
		interval time.Duration // execution interval
		elapse   int64         // total elapse time
		closed   int32         // is timer closed
		counter  int           // counter
	}
)

func init() {
	timerManager.timers = map[int64]*Timer{}
	timerManager.chClosingTimer = make(chan int64, timerBacklog)
	timerManager.chCreatedTimer = make(chan *Timer, timerBacklog)
}

// ID returns id of current timer
func (t *Timer) ID() int64 {
	return t.id
}

// Stop turns off a timer. After Stop, fn will not be called forever
func (t *Timer) Stop() {
	if atomic.LoadInt32(&t.closed) > 0 {
		return
	}
	atomic.StoreInt32(&t.closed, 1)
	timerManager.chClosingTimer <- t.id
}

// call job function with protection
func pjob(id int64, fn TimerFunc) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(fmt.Sprintf("Call timer funtion error, TimerID=%d, Error=%v\n%s", id, err, stack()))
		}
	}()

	fn()
}

// TODO: if closing timers'count in single cron call more than timerBacklog will case problem.
func cron() {
	now := time.Now().UnixNano()
	for id, t := range timerManager.timers {
		if t.createAt+t.elapse <= now {
			pjob(id, t.fn)
			t.elapse += int64(t.interval)

			// check timer counter
			if t.counter != loopForever && t.counter > 0 {
				t.counter--
				if t.counter == 0 {
					t.Stop()
				}
			}
		}
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
// The duration d must be greater than zero; if not, NewTimer will panic.
// Stop the timer to release associated resources.
func NewCountTimer(interval time.Duration, count int, fn TimerFunc) *Timer {
	if fn == nil {
		panic("nano/timer: nil timer function")
	}
	if interval <= 0 {
		panic("non-positive interval for NewTimer")
	}

	id := atomic.AddInt64(&timerManager.incrementId, 1)
	t := &Timer{
		id:       id,
		fn:       fn,
		createAt: time.Now().UnixNano(),
		interval: interval,
		elapse:   int64(interval), // first execution will be after interval
		counter:  count,
	}

	// add to manager
	timerManager.chCreatedTimer <- t
	return t
}

// SetTimerPrecision set the ticker precision, and time precision can not less
// than a Millisecond, and can not change after application running.
func SetTimerPrecision(precision time.Duration) {
	if precision < time.Millisecond {
		panic("time precision can not less than a Millisecond")
	}
	timerPrecision = precision
}
