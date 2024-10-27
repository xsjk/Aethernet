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

	SAMPLE_PER_BIT       = 50
	EXPECTED_TOTAL_BITS  = 10000
	EXPECTED_TOTAL_BYTES = EXPECTED_TOTAL_BITS / 8
	BIT_PER_FRAME        = 1000
	BYTE_PER_FRAME       = BIT_PER_FRAME / 8
	FRAME_INTERVAL       = 0

	AMPLITUDE  = 1.0
	ONE_FREQ   = 800
	ZERO_FREQ  = 800
	ONE_PHASE  = 0
	ZERO_PHASE = math.Pi

	POWER_THRESHOLD      = 4
	CORRECTION_THRESHOLD = 0.8
)

var preamble = modem.Float64ToInt32(modem.ChripConfig{
	MinFreq:    PREAMBLE_START_FREQ,
	MaxFreq:    PREAMBLE_END_FREQ,
	Length:     PREAMBLE_LENGTH,
	SampleRate: SAMPLE_RATE,
}.New())

var carriers = [2][]int32{
	modem.Float64ToInt32(
		modem.CarrierConfig{
			Amplitude:  AMPLITUDE,
			Freq:       ZERO_FREQ,
			Phase:      ZERO_PHASE,
			SampleRate: SAMPLE_RATE,
			Size:       SAMPLE_PER_BIT,
		}.New()),
	modem.Float64ToInt32(
		modem.CarrierConfig{
			Amplitude:  AMPLITUDE,
			Freq:       ONE_FREQ,
			Phase:      ONE_PHASE,
			SampleRate: SAMPLE_RATE,
			Size:       SAMPLE_PER_BIT,
		}.New()),
}

var BitModem = modem.NaiveBitModem{
	Preamble:                 preamble,
	BitPerFrame:              BIT_PER_FRAME,
	FrameInterval:            FRAME_INTERVAL,
	Carriers:                 carriers,
	DemodulatePowerThreshold: fixed.FromFloat(POWER_THRESHOLD),
	CorrectionThreshold:      fixed.FromFloat(CORRECTION_THRESHOLD),
}
