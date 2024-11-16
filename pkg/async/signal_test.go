package async

import (
	"testing"
	"time"
)

func TestSignal_Notify(t *testing.T) {
	s := make(Signal[struct{}])
	s.Notify()

	select {
	case <-s:
		// Success
	default:
		t.Error("Expected signal to be closed")
	}
}

func TestSignal_Await(t *testing.T) {
	var s Signal[struct{}]

	select {
	case <-s.Signal():
		t.Error("Expected channel to be open")
	default:
		// Success
	}
}

func TestSignal_NotifyAndAwait(t *testing.T) {
	var s Signal[int]

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.NotifyValue(42)
	}()

	select {
	case val := <-s.Signal():
		if val != 42 {
			t.Errorf("Expected 42 but got %d", val)
		}
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Error("Expected to receive signal within 200ms")
	}
}
