package timingwheel

import (
	"sync"
	"time"
)

type TimerTaskList struct {
	mu   *sync.Mutex
	head *TimerTaskEntry
	tail *TimerTaskEntry

	taskTimer *time.Timer
}

func newTimerTaskList(tw *TimingWheel) TimerTaskList {
	timer := time.NewTimer(time.Hour)
	// create and stop
	if !timer.Stop() {
		<-timer.C
	}

	var mu sync.Mutex
	bucket := TimerTaskList{
		taskTimer: timer,
		mu:        &mu,
	}

	go func() {
		for {
			select {
			case <-timer.C:
				bucket.drain(tw)
			case <-tw.stopChan:
				if !timer.Stop() {
					<-timer.C
				}
				return
			}
		}
	}()

	return bucket
}

func (l *TimerTaskList) add(entry *TimerTaskEntry) {
	if entry == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.tail == nil {
		l.head = entry
		l.tail = entry
		// wait for the timer to trigger
		l.taskTimer.Reset(time.Duration(entry.expiration-time.Now().UnixNano()/1000) * time.Millisecond)
	} else {
		l.tail.next = entry
		l.tail = entry
	}
}

func (l *TimerTaskList) drain(baseWheel *TimingWheel) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for l.head != nil {
		baseWheel.add(l.head)
		l.head = l.head.next
	}
}
