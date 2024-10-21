package config

import (
	"Aethernet/pkg/fixed"
	"Aethernet/pkg/modem"
	"math"
)

const (
	PREAMBLE_LENGTH     = 200
	PREAMBLE_START_FREQ = 6000.0
	PREAMBLE_END_FREQ   = 12000.0
	SAMPLE_RATE         = 48000.0

	SAMPLE_PER_BIT      = 50
	EXPECTED_TOTAL_BITS = 10000
	BIT_PER_FRAME       = 1000
	FRAME_INTERVAL      = 0

	AMPLITUDE  = 1.0
	ONE_FREQ   = 800
	ZERO_FREQ  = 800
	ONE_PHASE  = 0
	ZERO_PHASE = math.Pi

	POWER_THRESHOLD      = 4
	CORRECTION_THRESHOLD = 0.8
)

var Modem = modem.NaiveBitModem{
	Preamble: modem.Float64ToInt32(modem.PreambleParams{
		MinFreq:    PREAMBLE_START_FREQ,
		MaxFreq:    PREAMBLE_END_FREQ,
		Length:     PREAMBLE_LENGTH,
		SampleRate: SAMPLE_RATE,
	}.New()),
	BitPerFrame:   BIT_PER_FRAME,
	FrameInterval: FRAME_INTERVAL,
	CRCChecker:    modem.MakeCRC8Checker(0x07),
	Carriers: [2][]int32{
		modem.Float64ToInt32(
			modem.CarrierParams{
				Amplitude:  AMPLITUDE,
				Freq:       ZERO_FREQ,
				Phase:      ZERO_PHASE,
				SampleRate: SAMPLE_RATE,
				Size:       SAMPLE_PER_BIT,
			}.New()),
		modem.Float64ToInt32(
			modem.CarrierParams{
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
