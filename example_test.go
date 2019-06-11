package timingwheel_test

import (
	"fmt"

	"github.com/EagleChen/timingwheel"
)

func ExampleTimingWheel() {
	tw := timingwheel.NewTimingWheel()

	done := make(chan struct{})
	tw.Add(1, func() {
		fmt.Println("hello world")
		done <- struct{}{}
	})
	<-done
	// Output: hello world
}
