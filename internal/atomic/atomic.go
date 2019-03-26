package atomic

import "sync/atomic"

// IncWrapInt64 atomically increments a 64-bit signed integer, wrapping around zero.
//
// Specifically if p points to a value of math.MaxInt64 the result of calling
// IncWrapInt64 will be 0.
func IncWrapInt64(p *int64) int64 {
	for {
		o := atomic.LoadInt64(p)
		n := o + 1

		if n < 0 {
			n = 0
		}

		if atomic.CompareAndSwapInt64(p, o, n) {
			return n
		}
	}
}
