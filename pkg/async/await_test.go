package async

import (
	"testing"
	"time"
)

func TestAwait(t *testing.T) {
	f1 := Promise(func() int {
		time.Sleep(100 * time.Millisecond)
		return 1
	})
	f2 := Promise(func() int {
		time.Sleep(200 * time.Millisecond)
		return 2
	})
	f3 := Promise(func() int {
		time.Sleep(300 * time.Millisecond)
		return 3
	})

	startTime := time.Now()
	r1, r2, r3 := Await3(Gather3(f1, f2, f3))
	elapsedTime := time.Since(startTime)
	t.Logf("elapsed time: %v", elapsedTime)

	if r1 != 1 || r2 != 2 || r3 != 3 {
		t.Errorf("expected 1, 2, 3 but got %d, %d, %d", r1, r2, r3)
	}
}
