package device

import (
	"reflect"
	"testing"
	"time"
)

func TestLoopback(t *testing.T) {

	lastOutput := alloci32(BufferSize)

	var dev Device = &Loopback{
		SampleRate: 48000,
	}

	dev.Start(func(in, out []int32) {
		t.Logf("dev - in: %p, out: %p\n", in, out)
		if !reflect.DeepEqual(in, lastOutput) {
			t.Errorf("Expected %v, but got %v", lastOutput, in[0])
		}

		randi32(out)
		copy(lastOutput, out)
	})

	time.Sleep(time.Millisecond)
	dev.Stop()
}
