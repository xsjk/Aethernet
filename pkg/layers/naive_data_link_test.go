package layers

import (
	"Aethernet/pkg/device"
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/modem"
	"crypto/rand"
	"reflect"
	"testing"
	"time"
)

func TestNaiveDataLinkLayer(t *testing.T) {

	const (
		SAMPLE_RATE = 48000

		PHYSICAL_BYTE_PER_FRAME = 125 + 3
		FRAME_INTERVAL          = 256
		CARRIER_SIZE            = 2

		INPUT_BUFFER_SIZE             = 10000
		OUTPUT_BUFFER_SIZE            = 10
		PHYSICAL_RECEIVE_BUFFER_SIZE  = 10
		DATA_LINK_RECEIVE_BUFFER_SIZE = 0

		POWER_THRESHOLD = 30

		POWER_MONITOR_THRESHOLD = 0.4
		POWER_MONITOR_WINDOW    = 10

		ACK_TIMEOUT        = 1000 * time.Millisecond
		MAX_RETRY_ATTEMPTS = 5

		MIN_BACKOFF = 0
		MAX_BACKOFF = 100 * time.Millisecond
	)

	var preamble = modem.DigitalChripConfig{N: 4, Amplitude: 0x7fffffff}.New()

	var layers [2]NaiveDataLinkLayer

	network := device.Network[string]{
		Config: device.NetworkConfig[string]{
			{In: "w", Out: "w"},
			{In: "w", Out: "w"},
		},
		SampleRate: SAMPLE_RATE,
	}
	devices := network.Build()
	addresses := [2]byte{0x01, 0x02}

	for i := range layers {

		layers[i] = NaiveDataLinkLayer{
			PhysicalLayer: PhysicalLayer{
				Device: devices[i],
				Decoder: Decoder{
					Demodulator: modem.Demodulator{
						Preamble:                 preamble,
						CarrierSize:              CARRIER_SIZE,
						DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
						BufferSize:               PHYSICAL_RECEIVE_BUFFER_SIZE,
					},
					BufferSize: INPUT_BUFFER_SIZE,
				},
				Encoder: Encoder{
					Modulator: modem.Modulator{
						Preamble:      preamble,
						CarrierSize:   CARRIER_SIZE,
						BytePerFrame:  PHYSICAL_BYTE_PER_FRAME,
						FrameInterval: FRAME_INTERVAL,
					},
					BufferSize: OUTPUT_BUFFER_SIZE,
				},
				PowerMonitor: PowerMonitor{
					Threshold:  fixed.FromFloat(POWER_MONITOR_THRESHOLD),
					WindowSize: POWER_MONITOR_WINDOW,
				},
			},
			BufferSize: DATA_LINK_RECEIVE_BUFFER_SIZE,
			Address:    addresses[i],
		}
	}
	layers[0].Open()
	layers[1].Open()
	defer layers[0].Close()
	defer layers[1].Close()

	packet := make([]byte, 125)
	rand.Read(packet)

	sendDone := layers[0].SendAsync(packet)
	layers[0].SendAsync(packet)
	recvDone := layers[0].ReceiveAsync()

	<-sendDone

	time.Sleep(100 * time.Millisecond)

	select {
	case <-recvDone:
		t.Error("Should not receive any data")
	default:
		t.Log("No data received")
	}

	received := layers[1].Receive()
	if reflect.DeepEqual(received, packet) {
		t.Log("Received packet matches the sent packet")
	} else {
		t.Error("Received packet does not match the sent packet")
	}

}
