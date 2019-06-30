package timingwheel

import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"
)

const maxLevel = 7

// TimingWheel is an implementation of Hierarchical Timing Wheels.
// When lots of timers(about 1m) need to be maintained, use TimingWheel.
// Or else, just use timers
type TimingWheel struct {
	level       uint8
	wheelSize   int64
	tickMS      int64
	interval    int64
	currentTime int64
	buckets     []*TimerTaskList

	overflowWheel unsafe.Pointer // *TimingWheel, but may be nil

	wheelTimer *WheelTimer
}

// newTimingWheel creates a timingwheel with default config
func newTimingWheel() TimingWheel {
	return newTimingWheelWithConfig(defaultWheelSize, defaultTickMS)
}

func newTimingWheelWithConfig(wheelSize int64, tickMS int64) TimingWheel {
	return newTimingWheelWithConfigWithTimer(wheelSize, defaultTickMS, 0, nil)
}

// newTimingWheelWithConfig creates a timingwheel with customized config
func newTimingWheelWithConfigWithTimer(wheelSize int64, tickMS int64,
	level uint8, wheelTimer *WheelTimer) TimingWheel {
	timeMS := time.Now().UnixNano() / 1000000

	tw := TimingWheel{
		wheelSize:   wheelSize,
		tickMS:      tickMS,
		interval:    tickMS * wheelSize,
		currentTime: timeMS - (timeMS % tickMS),
		level:       level,
		wheelTimer:  wheelTimer,
	}

	buckets := make([]*TimerTaskList, wheelSize)
	for i := range buckets {
		idx := int(level)*int(wheelSize) + i
		buckets[i] = newTimerTaskList(idx)
		if level != 0 {
			buckets[i].start(wheelTimer)
		}
	}
	tw.buckets = buckets

	return tw
}

func (tw *TimingWheel) advanceClock(timeMS int64) {
	// for high level wheel, this `if` may often be false
	if timeMS >= tw.currentTime+tw.tickMS {
		tw.currentTime = timeMS - timeMS%tw.tickMS

		wheel := atomic.LoadPointer(&tw.overflowWheel)
		if wheel != nil {
			(*TimingWheel)(wheel).advanceClock(timeMS)
		}
	}
}

func (tw *TimingWheel) addEntry(entry *TimerTaskEntry, stopping bool) error {
	if entry.expiration < tw.currentTime+tw.tickMS { // fire now
		// TODO: dispatch action
		go entry.action()
	} else if entry.expiration < tw.currentTime+tw.interval { // add to current wheel
		actualIdx := (entry.expiration / tw.tickMS) % tw.wheelSize
		tw.buckets[actualIdx].add(entry, tw.tickMS)
	} else { // add to higher level wheel
		if tw.level+1 >= maxLevel {
			return errors.New("too many levels of timing wheel")
		}
		wheel := atomic.LoadPointer(&tw.overflowWheel)
		if wheel == nil {
			if stopping {
				return errors.New("stopping, no more task in higher level")
			}
			timingWheel := newTimingWheelWithConfigWithTimer(tw.wheelSize, tw.interval,
				tw.level+1, tw.wheelTimer)
			atomic.CompareAndSwapPointer(&tw.overflowWheel, nil, unsafe.Pointer(&timingWheel))
			wheel = atomic.LoadPointer(&tw.overflowWheel)
		}

		return (*TimingWheel)(wheel).addEntry(entry, stopping)
	}

	return nil
}

func (tw *TimingWheel) stop() {
	if wheel := atomic.LoadPointer(&tw.overflowWheel); wheel != nil {
		(*TimingWheel)(wheel).stop()
	}

	for i := range tw.buckets {
		tw.buckets[i].stop()
	}
}
