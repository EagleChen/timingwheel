package timingwheel_test

import (
	"fmt"
	"time"

	"github.com/EagleChen/timingwheel"
)

func ExampleTimingWheel() {
	tw := timingwheel.NewWheelTimer()
	tw.Start()

	done := make(chan struct{})
	if err := tw.Add(time.Now().UnixNano()/1000000+1, func() {
		fmt.Println("hello world")
		done <- struct{}{}
	}); err != nil {
		fmt.Println(err)
		done <- struct{}{}
	}
	<-done
	if err := tw.After(5, func() {
		fmt.Println("hello world again")
		done <- struct{}{}
	}); err != nil {
		fmt.Println(err)
		done <- struct{}{}
	}
	<-done

	// create another timewheel internally
	if err := tw.After(30, func() {
		fmt.Println("hello world again and again")
		done <- struct{}{}
	}); err != nil {
		done <- struct{}{}
	}
	<-done

	tw.After(10, func() {
		fmt.Println("won't run")
	})
	tw.Stop()
	// Output:
	// hello world
	// hello world again
	// hello world again and again
}
