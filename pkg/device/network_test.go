package device

import (
	"reflect"
	"testing"
	"time"
)

func TestNetwork(t *testing.T) {

	lastOutSum1 := alloci32(BufferSize)
	lastOutSum2 := alloci32(BufferSize)
	lastOutSum3 := alloci32(BufferSize)
	lastOutSum4 := alloci32(BufferSize)
	outputSum1 := alloci32(BufferSize)
	outputSum3 := alloci32(BufferSize)
	outputSum4 := alloci32(BufferSize)

	network := Network[string]{
		SampleRate: 48000,
		Config: NetworkConfig[string]{
			{In: "buf1", Out: "buf1"},
			{In: "buf1", Out: "buf1"},
			{In: "buf2", Out: "buf2"},
			{In: "buf3", Out: "buf4"},
			{In: "buf4", Out: "buf3"},
		},
		LateUpdate: func() {
			copy(lastOutSum1, outputSum1)
			copy(lastOutSum3, outputSum3)
			copy(lastOutSum4, outputSum4)
			cleari32(outputSum1)
		},
	}

	devs := network.Build()

	devs[0].Start(func(in, out []int32) {

		t.Logf("[dev1] - in: %p, out: %p\n", in, out)
		if !reflect.DeepEqual(in, lastOutSum1) {
			t.Errorf("[dev1] Expected %v, but got %v", lastOutSum1, in)
		}

		randi32(out)
		// t.Logf("[dev1] - out: %v\n", out)
		sumi32(outputSum1, out, outputSum1)
	})

	devs[1].Start(func(in, out []int32) {
		t.Logf("[dev2] - in: %p, out: %p\n", in, out)

		if !reflect.DeepEqual(in, lastOutSum1) {
			t.Errorf("[dev2] Expected %v, but got %v", lastOutSum1, in)
		}

		randi32(out)
		// t.Logf("[dev2] - out: %v\n", out)
		sumi32(outputSum1, out, outputSum1)
	})

	devs[2].Start(func(in, out []int32) {
		t.Logf("[dev3] - in: %p, out: %p\n", in, out)
		if !reflect.DeepEqual(in, lastOutSum2) {
			t.Errorf("[dev3] Expected %v, but got %v", lastOutSum2, in[0])
		}

		randi32(out)
		// t.Logf("[dev3] - out: %v\n", out)
		copy(lastOutSum2, out)
	})

	devs[3].Start(func(in, out []int32) {

		t.Logf("[dev4] - in: %p, out: %p\n", in, out)
		if !reflect.DeepEqual(in, lastOutSum4) {
			t.Errorf("[dev4] Expected %v, but got %v", lastOutSum4, in)
		}

		randi32(out)
		// t.Logf("[dev4] - out: %v\n", out)
		copy(outputSum3, out)
	})

	devs[4].Start(func(in, out []int32) {

		t.Logf("[dev5] - in: %p, out: %p\n", in, out)
		if !reflect.DeepEqual(in, lastOutSum3) {
			t.Errorf("[dev5] Expected %v, but got %v", lastOutSum3, in)
		}
		randi32(out)
		// t.Logf("[dev5] - out: %v\n", out)
		copy(outputSum4, out)
	})

	time.Sleep(3 * time.Millisecond)

	network.Stop()

}
