package layer

import (
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/modem"
	"crypto/rand"
	"reflect"
	"testing"
	"time"
)

func TestMACLayer(t *testing.T) {

	const (
		BYTE_PER_FRAME = 125
		FRAME_INTERVAL = 10
		CARRIER_SIZE   = 3
		INTERVAL_SIZE  = 10
		PAYLOAD_SIZE   = 32

		INPUT_BUFFER_SIZE  = 10000
		OUTPUT_BUFFER_SIZE = 1

		POWER_THRESHOLD      = 30
		CORRECTION_THRESHOLD = 0.8
	)

	var preamble = modem.DigitalChripConfig{N: 4, Amplitude: 0x7fffffff}.New()

	var layers [2]MACLayer
	var devices [2]Device
	var addresses [2]MACAddress = [2]MACAddress{0x0, 0x1}

	devices[0], devices[1] = (&CrossfeedDeviceManager{SampleRate: 48000}).Generate()

	for i := range layers {
		layers[i] = MACLayer{
			PhysicalLayer: PhysicalLayer{
				Device: devices[i],
				Decoder: Decoder{
					Demodulator: modem.Demodulator{
						Preamble:                 preamble,
						CarrierSize:              CARRIER_SIZE,
						CorrectionThreshold:      fixed.FromFloat(CORRECTION_THRESHOLD),
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
			},
			Address:    addresses[i],
			ACKTimeout: 100 * time.Millisecond,
			OutputChan: make(chan []byte),
		}
	}

	layers[0].Open()
	layers[1].Open()

	inputBytes := make([]byte, 1000)
	rand.Read(inputBytes)

	go layers[0].Send(addresses[1], inputBytes)

	output := layers[1].Receive()

	t.Logf("len(inputBytes) = %d, len(output) = %d", len(inputBytes), len(output))
	if !reflect.DeepEqual(inputBytes, output) {
		t.Errorf("inputBytes and outputBytes are different")
	}

	layers[0].Close()
	layers[1].Close()
}
