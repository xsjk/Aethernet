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
