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

	// go func() {
	// 	for {
	// 		select {
	// 		case <-timer.C:
	// 			bucket.drain(tw)
	// 		case <-bucket.stopChan:
	// 			bucket.mu.Lock()
	// 			defer bucket.mu.Unlock()
	// 			if !timer.Stop() {
	// 				<-timer.C
	// 			}
	// 			return
	// 		}
	// 	}
	// }()

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

func (l *TimerTaskList) drain(baseWheel *TimingWheel) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for l.head != nil {
		baseWheel.addEntry(l.head)
		l.head = l.head.next
	}
	l.tail = nil // reset tail
	l.timerSetted = false
}

func (l *TimerTaskList) start(w *WheelTimer) {
	go func() {
		w.waitGroup.Add(1)
		for {
			select {
			case <-l.taskTimer.C:
				w.drainBucketChan <- l
			case <-l.stopChan:
				l.mu.Lock()
				defer l.mu.Unlock()
				if !l.taskTimer.Stop() && l.timerSetted {
					<-l.taskTimer.C
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
