package timingwheel

import (
	"sync"
	"time"
)

const (
	defaultWheelSize = 20
	defaultTickMS    = 1

	defaultChanBuffer = 100
)

// WheelTimer is the timer wrapper for timing wheel
type WheelTimer struct {
	wheel TimingWheel // the base timing wheel

	waitGroup *sync.WaitGroup
	clockLock *sync.RWMutex

	updateClockChan  chan *TimerTaskList
	drainBucketChan  chan []*TimerTaskEntry
	stopClockChan    chan struct{}
	stopDrainingChan chan struct{}

	drainBuckets []*TimerTaskEntry
}

// NewWheelTimer creates a WheelTimer with default configs.
func NewWheelTimer() *WheelTimer {
	return NewWheelTimerWithConfig(defaultWheelSize, defaultTickMS)
}

// NewWheelTimerWithConfig creates a WheelTimer with given wheelSize and tickMs.
func NewWheelTimerWithConfig(wheelSize int64, tickMS int64) *WheelTimer {
	wheel := newTimingWheelWithConfig(wheelSize, tickMS)
	var wg sync.WaitGroup
	var m sync.RWMutex
	w := WheelTimer{
		wheel:            wheel,
		waitGroup:        &wg,
		clockLock:        &m,
		updateClockChan:  make(chan *TimerTaskList, defaultChanBuffer),
		drainBucketChan:  make(chan []*TimerTaskEntry, defaultChanBuffer),
		stopClockChan:    make(chan struct{}),
		stopDrainingChan: make(chan struct{}),
	}

	w.wheel.wheelTimer = &w
	return &w
}

// Start starts the wheel timer.
func (w *WheelTimer) Start() error {
	go func() { // handle 'advance clock'
		w.waitGroup.Add(1)
		for {
			// use readwrite mutex for advanceClock(write lock) and 'add'/'after'(read lock)
			// simutaneously add is ok(so use read lock in 'add')
			select {
			case bucket := <-w.updateClockChan: // for draining bucket timer
				w.clockLock.Lock()

				var heads []*TimerTaskEntry

				maxExpiration := bucket.expiration
				// keep 'drain' fast, so the write lock will be released soon
				if head := bucket.drain(); head != nil {
					heads = append(heads, head)
				}

				// bucket timer expiration may be fired out of order
				// so clean up all the remaining drainBucketChan(may be triggered by earlier timer)
			CleanupLoop:
				for {
					select {
					case otherBucket := <-w.updateClockChan:
						if maxExpiration < otherBucket.expiration {
							maxExpiration = otherBucket.expiration
						}
						if head := otherBucket.drain(); head != nil {
							heads = append(heads, head)
						}
					default: // no remaining timer, just break the cleanup for loop
						break CleanupLoop
					}
				}

				// only in this goroutine, advanceClock will be called
				w.wheel.advanceClock(maxExpiration)

				w.clockLock.Unlock()
				// after the lock is released
				w.drainBucketChan <- heads
			case <-w.stopClockChan:
				close(w.drainBucketChan)
				w.waitGroup.Done()
				return
			}
		}
	}()

	go func() { // handle bucket draining
		w.waitGroup.Add(1)
		for {
			select {
			case heads := <-w.drainBucketChan:
				for _, head := range heads {
					for head != nil {
						w.addEntry(head) // important: use read lock
						head = head.next
					}
				}
			case <-w.stopDrainingChan:
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

// Stop stops the wheel timer. Once stopped, it should not be started again.
func (w *WheelTimer) Stop() {
	w.wheel.stop()
	w.stopClockChan <- struct{}{}
	w.stopDrainingChan <- struct{}{}
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

func (w *WheelTimer) addEntry(entry *TimerTaskEntry) error {
	w.clockLock.RLock()
	defer w.clockLock.RUnlock()
	return w.wheel.addEntry(entry)
}
