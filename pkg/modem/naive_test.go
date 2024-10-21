package modem

import (
	"Aethernet/pkg/fixed"
	"math"
	"reflect"
	"testing"
)

// const (
// 	PREAMBLE_LENGTH     = 10000
// 	PREAMBLE_START_FREQ = 6000.0
// 	PREAMBLE_END_FREQ   = 12000.0
// 	SAMPLE_RATE         = 48000.0

// 	SAMPLE_PER_BIT      = 30
// 	EXPECTED_TOTAL_BITS = 1000
// 	BIT_PER_FRAME       = 1000
// 	FRAME_INTERVAL      = 0

// 	AMPLITUDE  = 1.0
// 	ONE_FREQ   = 800
// 	ZERO_FREQ  = 1400
// 	ONE_PHASE  = 0
// 	ZERO_PHASE = math.Pi

// 	POWER_THRESHOLD = 20
// )

const (
	PREAMBLE_LENGTH     = 10000
	PREAMBLE_START_FREQ = 6000.0
	PREAMBLE_END_FREQ   = 12000.0
	SAMPLE_RATE         = 48000.0

	SAMPLE_PER_BIT      = 30
	EXPECTED_TOTAL_BITS = 1000
	BIT_PER_FRAME       = 1000
	FRAME_INTERVAL      = 10

	AMPLITUDE  = 1.0
	ONE_FREQ   = 800
	ZERO_FREQ  = 1000
	ONE_PHASE  = 0
	ZERO_PHASE = math.Pi

	POWER_THRESHOLD = 20
)

var modem = NaiveModem{
	Preamble: Float64ToInt32(PreambleParams{
		MinFreq:    PREAMBLE_START_FREQ,
		MaxFreq:    PREAMBLE_END_FREQ,
		Length:     PREAMBLE_LENGTH,
		SampleRate: SAMPLE_RATE,
	}.New()),
	BitPerFrame:   BIT_PER_FRAME,
	FrameInterval: FRAME_INTERVAL,
	CRCChecker:    CRC8Checker{},
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
	CorrectionThreshold:      fixed.FromFloat(0.8),
}

func TestNaiveModem(t *testing.T) {

	inputBits := make([]bool, EXPECTED_TOTAL_BITS)
	for i := 0; i < EXPECTED_TOTAL_BITS; i++ {
		inputBits[i] = i%2 == 1
	}

	modulatedData := modem.Modulate(inputBits)
	outputBits := modem.Demodulate(modulatedData)

	if !reflect.DeepEqual(inputBits, outputBits) {
		t.Errorf("inputBits and outputBits are different")
	}
}
