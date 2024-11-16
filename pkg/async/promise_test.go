package async

import (
	"testing"
	"time"
)

func TestPromise(t *testing.T) {
	expected := 42
	resultChan := Promise(func() int {
		time.Sleep(100 * time.Millisecond)
		return expected
	})

	select {
	case result := <-resultChan:
		if result != expected {
			t.Fatalf("Expected %d but got %d", expected, result)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("TestAwaitResult timed out")
	}
}
