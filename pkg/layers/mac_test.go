package layers

import (
	"Aethernet/pkg/device"
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/modem"
	"crypto/rand"
	"testing"
	"time"
)

func TestMACLayer(t *testing.T) {

	const (
		SAMPLE_RATE = 48000

		BYTE_PER_FRAME = 125
		FRAME_INTERVAL = 256
		CARRIER_SIZE   = 3
		INTERVAL_SIZE  = 10
		PAYLOAD_SIZE   = 32

		INPUT_BUFFER_SIZE  = 10000
		OUTPUT_BUFFER_SIZE = 1

		POWER_THRESHOLD = 30

		POWER_MONITOR_THRESHOLD = 0.4
		POWER_MONITOR_WINDOW    = 10

		ACK_TIMEOUT        = 1000 * time.Millisecond
		MAX_RETRY_ATTEMPTS = 5

		MIN_BACKOFF = 0
		MAX_BACKOFF = 100 * time.Millisecond
	)

	var preamble = modem.DigitalChripConfig{N: 4, Amplitude: 0x7fffffff}.New()

	var layers [2]MACLayer
	var addresses [2]MACAddress = [2]MACAddress{0x0, 0x1}

	network := device.Network[string]{
		Config: device.NetworkConfig[string]{
			{In: "w", Out: "w"},
			{In: "w", Out: "w"},
		},
		SampleRate: SAMPLE_RATE,
	}

	devices := network.Build()

	for i := range layers {
		layers[i] = MACLayer{
			PhysicalLayer: PhysicalLayer{
				Device: devices[i],
				Decoder: Decoder{
					Demodulator: modem.Demodulator{
						Preamble:                 preamble,
						CarrierSize:              CARRIER_SIZE,
						DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
						OutputChan:               make(chan []byte, 10),
					},
					BufferSize: INPUT_BUFFER_SIZE,
				},
				Encoder: Encoder{
					Modulator: modem.Modulator{
						Preamble:      preamble,
						CarrierSize:   CARRIER_SIZE,
						BytePerFrame:  BYTE_PER_FRAME,
						FrameInterval: FRAME_INTERVAL,
					},
					BufferSize: OUTPUT_BUFFER_SIZE,
				},
				PowerMonitor: PowerMonitor{
					Threshold:  fixed.FromFloat(POWER_MONITOR_THRESHOLD),
					WindowSize: POWER_MONITOR_WINDOW,
				},
			},
			Address:    addresses[i],
			ACKTimeout: ACK_TIMEOUT,
			MaxRetries: MAX_RETRY_ATTEMPTS,
			BackoffTimer: RandomBackoffTimer{
				MinDelay: MIN_BACKOFF,
				MaxDelay: MAX_BACKOFF,
			},
			OutputChan: make(chan []byte, 10),
		}
	}

	layers[0].Open()
	layers[1].Open()

	packet0 := make([]byte, 6250)
	packet1 := make([]byte, 6250)
	rand.Read(packet0)
	rand.Read(packet1)

	done0 := make(chan bool)
	done1 := make(chan bool)

	var start_time, finish_time0, finish_time1 time.Time

	start_time = time.Now()

	go func() {
		err := layers[0].Send(addresses[1], packet0)
		if err != nil {
			t.Errorf("Error sending packet0: %v", err)
		}
		done0 <- true
		finish_time0 = time.Now()
	}()
	go func() {
		err := layers[1].Send(addresses[0], packet1)
		if err != nil {
			t.Errorf("Error sending packet1: %v", err)
		}
		done1 <- true
		finish_time1 = time.Now()
	}()

	<-done0
	<-done1

	t.Logf("Time taken to send packets: %v", finish_time0.Sub(start_time))
	t.Logf("Time taken to send packets: %v", finish_time1.Sub(start_time))

	// output1, err := layers[1].ReceiveWithTimeout(2 * time.Second)
	// if err != nil {
	// 	t.Errorf("Error receiving packet1: %v", err)
	// } else {
	// 	<-done0
	// 	t.Logf("len(packet0) = %d, len(output1) = %d", len(packet0), len(output1))
	// 	if !reflect.DeepEqual(packet0, output1) {
	// 		t.Errorf("packet0 and output1 are different")
	// 	} else {
	// 		t.Logf("packet0 and output1 are the same")
	// 	}
	// }
	// output0, err := layers[0].ReceiveWithTimeout(2 * time.Second)
	// if err != nil {
	// 	t.Errorf("Error receiving packet0: %v", err)
	// } else {
	// 	<-done1
	// 	t.Logf("len(packet1) = %d, len(output0) = %d", len(packet1), len(output0))
	// 	if !reflect.DeepEqual(packet1, output0) {
	// 		t.Errorf("packet1 and output0 are different")
	// 	} else {
	// 		t.Logf("packet1 and output0 are the same")
	// 	}
	// }

	layers[0].Close()
	layers[1].Close()
}
