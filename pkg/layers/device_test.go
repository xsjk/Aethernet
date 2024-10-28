package layer

import (
	"reflect"
	"testing"
	"time"

	"golang.org/x/exp/rand"
)

func TestLoopbackDevice(t *testing.T) {

	lastOutput := make([]int32, 512)

	var dev Device = &LoopbackDevice{
		SampleRate: 48000,
	}

	dev.Start(func(in, out [][]int32) {
		t.Logf("dev - in: %p, out: %p\n", in, out)
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

func TestCrossfeedDevice(t *testing.T) {

	var lastOutputs [2][]int32
	lastOutputs[0] = make([]int32, 512)
	lastOutputs[1] = make([]int32, 512)

	var dev1, dev2 Device = (&CrossfeedDeviceManager{
		SampleRate: 48000,
	}).Generate()

	dev1.Start(func(in, out [][]int32) {
		t.Logf("dev1 - in: %p, out: %p\n", in, out)
		if !reflect.DeepEqual(in[0], lastOutputs[1]) {
			t.Errorf("Expected %v, but got %v", lastOutputs[1], in[0])
		}
		for i := range out[0] {
			out[0][i] = rand.Int31()
		}
		copy(lastOutputs[0], out[0])
	})

	dev2.Start(func(in, out [][]int32) {
		t.Logf("dev2 - in: %p, out: %p\n", in, out)
		if !reflect.DeepEqual(in[0], lastOutputs[0]) {
			t.Errorf("Expected %v, but got %v", lastOutputs[0], in[0])
		}
		for i := range out[0] {
			out[0][i] = rand.Int31()
		}
		copy(lastOutputs[1], out[0])
	})

	time.Sleep(time.Millisecond)

	dev1.Stop()
	dev2.Stop()

}
