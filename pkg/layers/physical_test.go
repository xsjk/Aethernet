package layer

import (
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/modem"
	"testing"
)

func TestPhysicalLayer(t *testing.T) {

	const (
		BYTE_PER_FRAME = 125
		FRAME_INTERVAL = 10
		CARRIER_SIZE   = 3
		INTERVAL_SIZE  = 10
		PAYLOAD_SIZE   = 32

		POWER_THRESHOLD      = 30
		CORRECTION_THRESHOLD = 0.8
	)

	var preamble = modem.DigitalChripConfig{N: 4, Amplitude: 0x7fffffff}.New()

	var physicalLayer = PhysicalLayer{
		device: &LoopbackDevice{},
		decoder: Decoder{
			demodulator: modem.Demodulator{
				Preamble:                 preamble,
				CarrierSize:              CARRIER_SIZE,
				CorrectionThreshold:      fixed.FromFloat(CORRECTION_THRESHOLD),
				DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
				OutputChan:               make(chan []byte, 10),
			},
		},
		encoder: Encoder{
			modulator: modem.Modulator{
				Preamble:      preamble,
				CarrierSize:   CARRIER_SIZE,
				BytePerFrame:  BYTE_PER_FRAME,
				FrameInterval: FRAME_INTERVAL,
			},
			outputBuffer: make(chan []int32, 10),
		},
	}

	physicalLayer.Open()

	physicalLayer.Send([]byte("Hello, World!"))

	output := physicalLayer.Receive()

	if string(output) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', but got '%s'", string(output))
	} else {
		t.Logf("Received: %s", string(output))
	}

	physicalLayer.Close()
}
