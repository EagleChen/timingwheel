package timingwheel

import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	defaultWheelSize   = 20
	defaultTickMS      = 1
	defaultWorkBufSize = 100

	maxLevel = 7
)

type TimingWheel struct {
	level       uint8
	wheelSize   int64
	tickMS      int64
	interval    int64
	currentTime int64
	buckets     []TimerTaskList
	baseWheel   *TimingWheel

	workChan chan func()
	stopChan chan struct{}

	overflowWheel unsafe.Pointer // *TimingWheel, but may be nil
}

// NewTimingWheel creates a timingwheel with default config
func NewTimingWheel() TimingWheel {
	return NewTimingWheelWithConfig(defaultWheelSize, defaultTickMS, defaultWorkBufSize)
}

func NewTimingWheelWithConfig(wheelSize int64, tickMS int64, workBufSize int) TimingWheel {
	return NewTimingWheelWithConfigWithBaseWheel(defaultWheelSize, defaultTickMS, defaultWorkBufSize, 0, nil)
}

// NewTimingWheelWithConfig creates a timingwheel with customized config
func NewTimingWheelWithConfigWithBaseWheel(wheelSize int64, tickMS int64, workBufSize int,
	level uint8, baseWheel *TimingWheel) TimingWheel {
	timeMS := time.Now().UnixNano() / 1000

	tw := TimingWheel{
		wheelSize:   wheelSize,
		tickMS:      tickMS,
		interval:    tickMS * wheelSize,
		currentTime: timeMS - (timeMS % tickMS),
	}

	if baseWheel == nil { // tw is base timingwheel
		baseWheel = &tw
		tw.workChan = make(chan func(), workBufSize)
		tw.stopChan = make(chan struct{})
	} else {
		tw.workChan = baseWheel.workChan
		tw.stopChan = baseWheel.stopChan
	}

	tw.baseWheel = baseWheel
	buckets := make([]TimerTaskList, wheelSize)
	for i := range buckets {
		buckets[i] = newTimerTaskList(baseWheel)
	}
	tw.buckets = buckets

	return tw
}

func (tw *TimingWheel) addOverflowTimingWheel() error {
	if tw.level+1 >= maxLevel {
		return errors.New("too many levels of timing wheel")
	}

	// no need to use mutex here
	wheel := atomic.LoadPointer(&tw.overflowWheel)
	if wheel == nil {
		timingWheel := NewTimingWheelWithConfigWithBaseWheel(tw.wheelSize, tw.interval, 0,
			tw.level+1, tw.baseWheel)
		atomic.CompareAndSwapPointer(&tw.overflowWheel, nil, unsafe.Pointer(&timingWheel))
	}

	return nil
}

// TODO: no race issue?
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

// Add adds timer task entry to timing wheel or executes the entry
// expiration in milli seconds
func (tw *TimingWheel) Add(expiration int64, action func()) error {
	entry := TimerTaskEntry{expiration: expiration, action: action}
	return tw.add(&entry)
}

func (tw *TimingWheel) add(entry *TimerTaskEntry) error {
	if entry.expiration < tw.currentTime+tw.tickMS { // fire now
		// dispatch action
		tw.workChan <- entry.action
	} else if entry.expiration < tw.currentTime+tw.interval { // add to current wheel
		actualIdx := (entry.expiration / tw.tickMS) % tw.wheelSize
		tw.buckets[actualIdx].add(entry)
	} else { // add to higher level wheel
		wheel := atomic.LoadPointer(&tw.overflowWheel)
		if wheel == nil {
			if err := tw.addOverflowTimingWheel(); err != nil {
				return err
			}
			wheel = atomic.LoadPointer(&tw.overflowWheel)
		}

		(*TimingWheel)(wheel).add(entry)
	}

	return nil
}
