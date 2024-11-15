package async

import (
	"testing"
	"time"
)

func TestAwait(t *testing.T) {
	done := Await(func() {
		time.Sleep(100 * time.Millisecond)
	})

	select {
	case <-done:
		// Test passed
	case <-time.After(200 * time.Millisecond):
		t.Fatal("TestAwait timed out")
	}
}

func TestAwaitResult(t *testing.T) {
	expected := 42
	resultChan := AwaitResult(func() int {
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
