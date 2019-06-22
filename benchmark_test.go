package timingwheel_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/EagleChen/timingwheel"
)

func genD(i int) time.Duration {
	return time.Duration(i%10000) * time.Millisecond
}

func BenchmarkTimingWheel_StartStop(b *testing.B) {
	t := timingwheel.NewWheelTimerWithConfig(2, 1)

	cases := []struct {
		name string
		N    int // the data size (i.e. number of existing timers)
	}{
		{"N-0", 0},
		// {"N-1m", 1000000},
		// {"N-5m", 5000000},
		// {"N-10m", 10000000},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			t.Start()
			for i := 0; i < c.N; i++ {
				t.After(rand.Int63n(4), func() {})
			}

			b.ResetTimer()
			for i := 0; i < 100; i++ {
				t.After(rand.Int63n(4), func() {})
			}
			b.StopTimer()
			t.Stop()
		})
	}
}

func BenchmarkStandardTimer_StartStop(b *testing.B) {
	cases := []struct {
		name string
		N    int // the data size (i.e. number of existing timers)
	}{
		{"N-0", 0},
		{"N-1m", 1000000},
		{"N-5m", 5000000},
		{"N-10m", 10000000},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			base := make([]*time.Timer, c.N)
			for i := 0; i < c.N; i++ {
				base[i] = time.AfterFunc(time.Duration(rand.Intn(40))*time.Millisecond, func() {})
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				time.AfterFunc(time.Duration(rand.Intn(40))*time.Millisecond, func() {}).Stop()
			}
			b.StopTimer()

			for i := 0; i < len(base); i++ {
				base[i].Stop()
			}
		})
	}
}
