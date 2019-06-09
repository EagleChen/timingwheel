package timingwheel

type TimerTaskEntry struct {
	expiration int64
	action     func()

	next *TimerTaskEntry
}
