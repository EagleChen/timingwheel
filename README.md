TimingWheel

A simple golang implementation of timimg wheel like kafka.

Example
see example_test.go for examples.

Benchmark
```
goarch: amd64
pkg: github.com/EagleChen/timingwheel
BenchmarkTimingWheel_StartStop/N-0-4 	              500000	      2450 ns/op
BenchmarkTimingWheel_StartStop/N-100k-4         	 1000000	      2189 ns/op
BenchmarkTimingWheel_StartStop/N-500k-4         	 1000000	      2351 ns/op
BenchmarkTimingWheel_StartStop/N-1m-4           	 1000000	      2218 ns/op
BenchmarkStandardTimer_StartStop/N-0-4          	 1000000	      1407 ns/op
BenchmarkStandardTimer_StartStop/N-100k-4       	 1000000	      1639 ns/op
BenchmarkStandardTimer_StartStop/N-500k-4       	 1000000	      1878 ns/op
BenchmarkStandardTimer_StartStop/N-1m-4         	  500000	      2889 ns/op
PASS
ok  	github.com/EagleChen/timingwheel	39.142s
```