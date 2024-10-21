package modem

import (
	"strings"
)

type BitSet struct {
	bits []uint64
	size int
}

func NewBitSet(size int) *BitSet {
	return &BitSet{
		bits: make([]uint64, (size+63)/64),
		size: size,
	}
}

func (b *BitSet) Set(pos int) {
	if pos >= b.size {
		return
	}
	b.bits[pos/64] |= 1 << (pos % 64)
}

func (b *BitSet) Clear(pos int) {
	if pos >= b.size {
		return
	}
	b.bits[pos/64] &^= 1 << (pos % 64)
}

func (b *BitSet) IsSet(pos int) bool {
	if pos >= b.size {
		return false
	}
	return b.bits[pos/64]&(1<<(pos%64)) != 0
}

func (b *BitSet) String() string {
	var sb strings.Builder
	for i := 0; i < b.size; i++ {
		if b.IsSet(i) {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return sb.String()
}

type BitSet8 byte

func (b *BitSet8) Set(pos int) {
	*b |= 1 << pos
}

func (b *BitSet8) Clear(pos int) {
	*b &^= 1 << pos
}

func (b *BitSet8) IsSet(pos int) bool {
	return *b&(1<<pos) != 0
}

func (b *BitSet8) String() string {
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		if b.IsSet(i) {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return sb.String()
}

func (b BitSet8) ToByte() byte {
	return byte(b)
}
