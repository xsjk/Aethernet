package modem

type Modem[T any] interface {
	Modulate(inputBytes []T) []int32
	Demodulate(inputSignal []int32) []T
}
