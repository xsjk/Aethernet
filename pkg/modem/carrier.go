package modem

import "math"

type CarrierConfig struct {
	Amplitude  float64
	Freq       float64
	Phase      float64
	SampleRate float64
	Size       int
}

func (p CarrierConfig) New() []float64 {
	signal := make([]float64, p.Size)
	for i := 0; i < p.Size; i++ {
		t := float64(i) / p.SampleRate
		signal[i] = p.Amplitude * math.Sin(2*math.Pi*p.Freq*t+p.Phase)
	}
	return signal
}
