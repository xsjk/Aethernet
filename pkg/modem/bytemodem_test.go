package modem

import (
	"Aethernet/pkg/fixed"
	"math"
	"reflect"
	"testing"
)

func TestNaiveByteModem(t *testing.T) {

	const (
		PREAMBLE_LENGTH     = 10000
		PREAMBLE_START_FREQ = 6000.0
		PREAMBLE_END_FREQ   = 12000.0
		SAMPLE_RATE         = 48000.0

		SAMPLE_PER_BIT       = 30
		EXPECTED_TOTAL_BYTES = 1000 / 8
		BYTE_PER_FRAME       = 1000 / 8
		FRAME_INTERVAL       = 10

		AMPLITUDE  = 1.0
		ONE_FREQ   = 800
		ZERO_FREQ  = 1000
		ONE_PHASE  = 0
		ZERO_PHASE = math.Pi

		POWER_THRESHOLD      = 20
		CORRECTION_THRESHOLD = 0.8
	)

	var modem = NaiveByteModem{
		Preamble: Float64ToInt32(PreambleParams{
			MinFreq:    PREAMBLE_START_FREQ,
			MaxFreq:    PREAMBLE_END_FREQ,
			Length:     PREAMBLE_LENGTH,
			SampleRate: SAMPLE_RATE,
		}.New()),
		BytePerFrame:  BYTE_PER_FRAME,
		FrameInterval: FRAME_INTERVAL,
		CRCChecker:    MakeCRC8Checker(0x07),
		Carriers: [2][]int32{
			Float64ToInt32(CarrierParams{
				Amplitude:  AMPLITUDE,
				Freq:       ZERO_FREQ,
				Phase:      ZERO_PHASE,
				SampleRate: SAMPLE_RATE,
				Size:       SAMPLE_PER_BIT,
			}.New()),
			Float64ToInt32(CarrierParams{
				Amplitude:  AMPLITUDE,
				Freq:       ONE_FREQ,
				Phase:      ONE_PHASE,
				SampleRate: SAMPLE_RATE,
				Size:       SAMPLE_PER_BIT,
			}.New()),
		},
		DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
		CorrectionThreshold:      fixed.FromFloat(CORRECTION_THRESHOLD),
	}

	inputBytes := make([]byte, EXPECTED_TOTAL_BYTES)
	for i := 0; i < EXPECTED_TOTAL_BYTES; i++ {
		inputBytes[i] = 0b01010101
	}

	modulatedData := modem.Modulate(inputBytes)
	outputBytes := modem.Demodulate(modulatedData)

	if !reflect.DeepEqual(inputBytes, outputBytes) {
		t.Errorf("inputBytes and outputBytes are different")
	}
}
