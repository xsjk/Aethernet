package layer

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/exp/rand"
)

func TestLoopbackDevice(t *testing.T) {

	lastOutput := make([]int32, 512)

	dev := &LoopbackDevice{
		SampleRate: 48000,
	}

	i := 0
	dev.Start(func(in, out [][]int32) {
		t.Logf("i = %d", i)
		i++
		if !reflect.DeepEqual(in[0], lastOutput) {
			t.Errorf("Expected %v, but got %v", lastOutput, in[0])
		}
		for i := range out[0] {
			out[0][i] = rand.Int31()
		}
		copy(lastOutput, out[0])
	})

	time.Sleep(time.Millisecond)
	dev.Stop()
}
