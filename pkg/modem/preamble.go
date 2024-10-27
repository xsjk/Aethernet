package modem

import "math"

func chirp(out *[]float64, startFreq, endFreq float64, length int, sampleRate float64) {
	c := (endFreq - startFreq) / (float64(length) / sampleRate)
	f0 := startFreq

	for i := 0; i < length; i++ {
		t := float64(i) / sampleRate
		o := math.Sin(2 * math.Pi * (c/2*t + f0) * t)
		*out = append(*out, o)
	}
}

type ChripConfig struct {
	MinFreq    float64
	MaxFreq    float64
	Length     int
	SampleRate float64
}

func (p ChripConfig) New() []float64 {
	preamble := make([]float64, 0, p.Length)

	chirp(&preamble, p.MinFreq, p.MaxFreq, p.Length/2, p.SampleRate)
	chirp(&preamble, p.MaxFreq, p.MinFreq, p.Length/2, p.SampleRate)

	return preamble
}

type DigitalChripConfig struct {
	N         int
	Amplitude int32
}

func (p DigitalChripConfig) New() []int32 {
	var preamble []int32

	var repeat = func(n int, v int32) {
		for range n {
			preamble = append(preamble, v)
		}
	}

	for i := 1; i <= p.N; i++ {
		repeat(i, p.Amplitude)
		repeat(i, -p.Amplitude)
	}

	for i := p.N; i >= 1; i-- {
		repeat(i, p.Amplitude)
		repeat(i, -p.Amplitude)
	}

	return preamble
}
