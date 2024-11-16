package async

import (
	"testing"
	"time"
)

func TestJob(t *testing.T) {
	done := Job(func() {
		time.Sleep(100 * time.Millisecond)
	})

	select {
	case <-done:
		// Test passed
	case <-time.After(200 * time.Millisecond):
		t.Fatal("TestAwait timed out")
	}
}
