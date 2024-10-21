package modem

// Convert []int32 to []float64
func Int32ToFloat64(input []int32) []float64 {
	output := make([]float64, len(input))
	for i, v := range input {
		output[i] = float64(v) / 0x7fffffff
	}
	return output
}

// Convert []float64 to []int32
func Float64ToInt32(input []float64) []int32 {
	output := make([]int32, len(input))
	for i, v := range input {
		output[i] = int32(v * 0x7fffffff)
	}
	return output
}

// Convert []bool to []byte
func BoolToByte(input []bool) []byte {
	output := make([]byte, (len(input)+7)/8)
	for i, v := range input {
		if v {
			output[i/8] |= 1 << (7 - i%8)
		}
	}
	return output
}

// Convert []byte to []bool
func ByteToBool(input []byte) []bool {
	output := make([]bool, len(input)*8)
	for i, v := range input {
		for j := 0; j < 8; j++ {
			output[i*8+j] = (v>>uint(7-j))&1 == 1
		}
	}
	return output
}
