package utils

import "math"

func chirp(out []float64, minFreq, maxFreq float64, preambleLength int, sampleRate float64) []float64 {

	c := (maxFreq - minFreq) / (float64(preambleLength) / sampleRate)
	f0 := minFreq

	for i := 0; i < preambleLength; i++ {
		t := float64(i) / sampleRate
		o := math.Sin(2 * math.Pi * (c/2*t + f0) * t)
		out = append(out, o)
	}

	return out
}

func GeneratePreamble(minFreq, maxFreq float64, preambleLength int, sampleRate float64) []float64 {

	preamble := make([]float64, 0, preambleLength)

	preamble = chirp(preamble, minFreq, maxFreq, preambleLength/2, sampleRate)
	preamble = chirp(preamble, maxFreq, minFreq, preambleLength/2, sampleRate)

	return preamble
}
