package atomic

import (
	"math"
	"sync"
	"testing"
)

func TestIncWrapInt64(t *testing.T) {
	const n = 1000
	const m = n - 1

	var x int64

	var wg sync.WaitGroup
	wg.Add(m)

	for i := 0; i < m; i++ {
		go func() {
			IncWrapInt64(&x)
			wg.Done()
		}()
	}

	wg.Wait()

	if y := IncWrapInt64(&x); y != n {
		t.Errorf("expected %d, got %d", n, y)
	}

	x = math.MaxInt64 - m
	wg.Add(m)

	for i := 0; i < m; i++ {
		go func() {
			IncWrapInt64(&x)
			wg.Done()
		}()
	}

	wg.Wait()

	if y := IncWrapInt64(&x); y != 0 {
		t.Errorf("expected %d, got %d", 0, y)
	}
}
