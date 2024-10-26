package modem

import "golang.org/x/exp/constraints"

type bitSet[T constraints.Integer] struct {
	Value T
}

func BitSet[T constraints.Integer](value T) bitSet[T] {
	return bitSet[T]{Value: value}
}

func (b *bitSet[T]) Set(bit int) {
	b.Value |= (1 << bit)
}

func (b *bitSet[T]) Clear(bit int) {
	b.Value &^= (1 << bit)
}

func (b bitSet[T]) IsSet(bit int) bool {
	return b.Value&(1<<bit) != 0
}

func (b bitSet[T]) ForEach(f func(bool), n int) {
	for i := 0; i < n; i++ {
		f((b.Value & (1 << uint(i))) != 0)
	}
}
