package modem

import "Aethernet/pkg/fixed"

// A special dot product function that
// takes two input vectors represented by two slices of int32 (fixed-point numbers with 31 fractional bits)
// and return a dot product result is represented by a fixed-point number with fixed.D fractional bits
func dotProduct(a, b []int32) fixed.T {
	s := int64(0)
	for i := range min(len(a), len(b)) {
		s += (int64(a[i]) * int64(b[i])) >> 31
	}
	return fixed.T(s >> fixed.N)
}
