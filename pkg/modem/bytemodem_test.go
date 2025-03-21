package modem

import (
	"Aethernet/pkg/fixed"
	"crypto/rand"
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

		POWER_THRESHOLD = 10
	)

	var preamble = DigitalChripConfig{N: 4, Amplitude: 0x7fffffff}.New()

	var modem = NaiveByteModem{
		Modulator: Modulator{
			Preamble:      preamble,
			CarrierSize:   CARRIER_SIZE,
			BytePerFrame:  BYTE_PER_FRAME,
			FrameInterval: FRAME_INTERVAL,
		},
		Demodulator: Demodulator{
			Preamble:                 preamble,
			CarrierSize:              CARRIER_SIZE,
			DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
		},
	}

	inputBytes := make([]byte, 1000)
	rand.Read(inputBytes)

	modulatedData := modem.Modulate(inputBytes)
	go modem.Demodulate(modulatedData)
	outputBytes := <-modem.Demodulator.ReceiveAsync()

	if !reflect.DeepEqual(inputBytes, outputBytes) {
		t.Errorf("inputBytes and outputBytes are different")
	}
}
