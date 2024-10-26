package modem

import (
	"testing"
)

func TestBitSet(t *testing.T) {
	tests := []struct {
		name      string
		initial   uint32
		setBits   []int
		clearBits []int
		expected  uint32
	}{
		{"Set and Clear bits", 0, []int{1, 3, 5}, []int{3}, 34},
		{"Set bits only", 0, []int{0, 2, 4}, []int{}, 21},
		{"Clear bits only", 255, []int{}, []int{0, 1, 2}, 248},
		{"No operations", 15, []int{}, []int{}, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := BitSet(tt.initial)
			for _, bit := range tt.setBits {
				bs.Set(bit)
			}
			for _, bit := range tt.clearBits {
				bs.Clear(bit)
			}
			if bs.Value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, bs.Value)
			}
		})
	}
}

func TestIsSet(t *testing.T) {
	bs := BitSet(uint32(10)) // 1010 in binary
	tests := []struct {
		bit      int
		expected bool
	}{
		{1, true},
		{3, true},
		{0, false},
		{2, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if result := bs.IsSet(tt.bit); result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestForEach(t *testing.T) {
	bs := BitSet(uint32(10)) // 1010 in binary
	expected := []bool{false, true, false, true}
	var result []bool

	bs.ForEach(func(bit bool) {
		result = append(result, bit)
	}, 4)

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("at index %d, expected %v, got %v", i, v, result[i])
		}
	}
}
