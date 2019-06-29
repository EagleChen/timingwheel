package timingwheel_test

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/EagleChen/timingwheel"
)

func TestWheelTimerStop(t *testing.T) {
	tw := timingwheel.NewWheelTimer()
	tw.Start()
	tw.Stop()
	// Stop finishes
}

func TestWheelTimerAddValid(t *testing.T) {
	tw := timingwheel.NewWheelTimer()
	tw.Start()
	called := 0
	done := make(chan struct{})
	tw.Add(time.Now().UnixNano()/1000000+1, func() {
		called += 1
		done <- struct{}{}
	})
	<-done
	if called != 1 {
		t.Error("timer not fired")
	}
	tw.After(1, func() {
		called += 1
		done <- struct{}{}
	})
	<-done
	if called != 2 {
		t.Error("timer not fired")
	}
	tw.Stop()
	// Stop finishes
}

func TestWheelTimerAddInValid(t *testing.T) {
	tw := timingwheel.NewWheelTimer()
	tw.Start()
	if err := tw.Add(time.Now().UnixNano()/1000000+8000000000000000000, func() {}); err == nil {
		t.Error("should return err if expiration is too large")
	}
	if err := tw.After(8000000000000000000, func() {}); err == nil {
		t.Error("should return err if expiration is too large")
	}
	tw.Stop()
	// Stop finishes
}

func TestWheelTimerWithLotsOfTimer(t *testing.T) {
	tw := timingwheel.NewWheelTimer()
	tw.Start()
	count := 100000
	hits := make([]bool, count)
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		// may use two or three levels
		expiration := time.Now().UnixNano()/1000000 + rand.Int63n(5)
		current_idx := i
		if err := tw.Add(expiration, func() {
			if hits[current_idx] {
				t.Error("some entry re-run")
			}
			hits[current_idx] = true
			wg.Done()
		}); err != nil {
			t.Error("should not return err")
			wg.Done()
		}
	}
	wg.Wait()
	tw.Stop()

	for i := 0; i < count; i++ {
		if !hits[i] {
			t.Error("some timer not fire")
		}
	}
}
