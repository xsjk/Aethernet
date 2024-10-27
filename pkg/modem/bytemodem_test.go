package modem

import (
	"Aethernet/pkg/fixed"
	"reflect"
	"testing"
)

func TestNaiveByteModem(t *testing.T) {

	const (
		BYTE_PER_FRAME = 125
		FRAME_INTERVAL = 10
		CARRIER_SIZE   = 3
		INTERVAL_SIZE  = 10
		PAYLOAD_SIZE   = 32

		POWER_THRESHOLD      = 10
		CORRECTION_THRESHOLD = 0.8
	)

	var preamble = DigitalChripConfig{N: 4, Amplitude: 0x7fffffff}.New()

	var outputChan = make(chan []byte, 10)

	var modem = NaiveByteModem{
		Modulator{
			Preamble:      preamble,
			CarrierSize:   CARRIER_SIZE,
			BytePerFrame:  BYTE_PER_FRAME,
			FrameInterval: FRAME_INTERVAL,
		},
		Demodulator{
			Preamble:                 preamble,
			CarrierSize:              CARRIER_SIZE,
			CorrectionThreshold:      fixed.FromFloat(CORRECTION_THRESHOLD),
			DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
			OutputChan:               outputChan,
		},
	}

	inputBytes := make([]byte, 1000)
	for i := range inputBytes {
		inputBytes[i] = 0b01010101
	}

	modulatedData := modem.Modulate(inputBytes)
	modem.Demodulate(modulatedData)
	outputBytes := <-outputChan

	if !reflect.DeepEqual(inputBytes, outputBytes) {
		t.Errorf("inputBytes and outputBytes are different")
	}
}
