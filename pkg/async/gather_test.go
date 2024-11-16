package async

import (
	"testing"
	"time"
)

func TestGather(t *testing.T) {
	f1 := Promise(func() int {
		time.Sleep(100 * time.Millisecond)
		return 1
	})
	f2 := Promise(func() string {
		time.Sleep(200 * time.Millisecond)
		return "two"
	})
	f3 := Promise(func() bool {
		time.Sleep(300 * time.Millisecond)
		return true
	})

	startTime := time.Now()
	r := <-Gather(f1, f2, f3)
	elapsedTime := time.Since(startTime)
	t.Logf("elapsed time: %v", elapsedTime)

	expected := []any{1, "two", true}
	for i, result := range r {
		if result != expected[i] {
			t.Errorf("expected %v, got %v", expected[i], result)
		}
	}
}

func TestGatherAutoCast(t *testing.T) {
	f1 := func() int {
		time.Sleep(100 * time.Millisecond)
		return 1
	}
	f2 := func() string {
		time.Sleep(200 * time.Millisecond)
		return "two"
	}
	f3 := func() bool {
		time.Sleep(300 * time.Millisecond)
		return true
	}

	startTime := time.Now()
	r := <-Gather(f1, f2, f3)
	elapsedTime := time.Since(startTime)
	t.Logf("elapsed time: %v", elapsedTime)

	expected := []any{1, "two", true}
	for i, result := range r {
		if result != expected[i] {
			t.Errorf("expected %v, got %v", expected[i], result)
		}
	}
}
