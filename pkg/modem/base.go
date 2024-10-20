package modem

type Modem interface {
	Modulate(inputBits []bool) []int32
	Demodulate(inputSignal []int32) []bool
}
