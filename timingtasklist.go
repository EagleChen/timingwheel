package timingwheel

import (
	"sync"
	"time"
)

type TimerTaskList struct {
	mu         *sync.Mutex
	head       *TimerTaskEntry
	tail       *TimerTaskEntry
	expiration int64 // milliseconds

	idx         int
	taskTimer   *time.Timer
	timerSetted bool
	stopChan    chan struct{}
}

func newTimerTaskList(idx int) *TimerTaskList {
	timer := time.NewTimer(time.Hour)
	// create and stop
	if !timer.Stop() {
		<-timer.C
	}

	var mu sync.Mutex
	bucket := TimerTaskList{
		idx:         idx,
		taskTimer:   timer,
		mu:          &mu,
		stopChan:    make(chan struct{}),
		timerSetted: false,
	}
	return &bucket
}

func (l *TimerTaskList) add(entry *TimerTaskEntry, wheelTickMS int64) {
	if entry == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.tail == nil {
		l.head = entry
		l.tail = entry
		// wait for the timer to trigger
		l.expiration = entry.expiration - entry.expiration%wheelTickMS
		d := time.Duration(l.expiration - time.Now().UnixNano()/1000000)
		l.taskTimer.Reset(d * time.Millisecond)
		l.timerSetted = true
	} else {
		l.tail.next = entry
		l.tail = entry
	}
}

// drain only returns head and reset itself
// it should not block too long
func (l *TimerTaskList) drain() *TimerTaskEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	head := l.head
	if l.head != nil {
		l.head = nil
		l.tail = nil // reset tail
		// l.timerSetted = false // the value changed in the task timer goroutine below
	}
	return head
}

func (l *TimerTaskList) start(w *WheelTimer) {
	go func() {
		w.waitGroup.Add(1)
		for {
			select {
			case <-l.taskTimer.C:
				w.updateClockChan <- l
				l.mu.Lock()
				l.timerSetted = false
				l.mu.Unlock()
			case <-l.stopChan:
				l.mu.Lock()
				defer l.mu.Unlock()
				if !l.taskTimer.Stop() {
					if l.timerSetted {
						<-l.taskTimer.C
					}
				}
				w.waitGroup.Done()
				return
			}
		}
	}()
}

func (l *TimerTaskList) stop() {
	if l.stopChan != nil {
		close(l.stopChan)
	}
}
