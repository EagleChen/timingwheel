package timingwheel

import (
	"sync"
	"time"
)

const (
	defaultWheelSize = 20
	defaultTickMS    = 1
)

type WheelTimer struct {
	wheel TimingWheel // the base timing wheel

	waitGroup *sync.WaitGroup
	clockLock *sync.RWMutex

	drainBucketChan chan *TimerTaskList
	stopChan        chan struct{}
}

func NewWheelTimer() *WheelTimer {
	return NewWheelTimerWithConfig(defaultWheelSize, defaultTickMS)
}

func NewWheelTimerWithConfig(wheelSize int64, tickMS int64) *WheelTimer {
	wheel := NewTimingWheelWithConfig(wheelSize, tickMS)
	var wg sync.WaitGroup
	var m sync.RWMutex
	w := WheelTimer{
		wheel:     wheel,
		waitGroup: &wg,
		clockLock: &m,
		// addBucketChan:   make(chan *TimerTaskList),
		drainBucketChan: make(chan *TimerTaskList),
		stopChan:        make(chan struct{}),
	}

	w.wheel.wheelTimer = &w
	return &w
}

func (w *WheelTimer) Start() error {
	go func() {
		w.waitGroup.Add(1)
		for {
			select {
			case bucket := <-w.drainBucketChan: // for draining bucket timer
				w.clockLock.Lock()
				// only in this goroutine, advanceClock will be called
				w.wheel.advanceClock(bucket.expiration)
				w.clockLock.Unlock()
				bucket.drain(&w.wheel)
			case <-w.stopChan:
				w.waitGroup.Done()
				return
			}
		}
	}()

	for i := range w.wheel.buckets {
		w.wheel.buckets[i].start(w)
	}

	return nil
}

func (w *WheelTimer) Stop() {
	w.wheel.stop()
	w.stopChan <- struct{}{}
	w.waitGroup.Wait()
}

// Add adds timer task entry to timing wheel or executes the entry
// expiration in milli seconds
func (w *WheelTimer) Add(expiration int64, action func()) error {
	w.clockLock.RLock()
	defer w.clockLock.RUnlock()
	entry := TimerTaskEntry{expiration: expiration, action: action}
	return w.wheel.addEntry(&entry)
}

// After adds timer task entry to timing wheel or executes the entry
// expiration is now + afterMS
func (w *WheelTimer) After(afterMS int64, action func()) error {
	w.clockLock.RLock()
	defer w.clockLock.RUnlock()

	entry := TimerTaskEntry{expiration: time.Now().UnixNano()/1000000 + afterMS, action: action}
	return w.wheel.addEntry(&entry)
}
