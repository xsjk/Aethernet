package fixed

// 32 = 1 + N + D
type T int32

const (
	D     = 6
	N     = 31 - D
	Denom = 1 << D

	Zero = T(0)
	One  = T(Denom)
)

func (f T) Add(other T) T {
	return T(f + other)
}

func (f T) Sub(other T) T {
	return T(f - other)
}

func (f T) Mul(other T) T {
	return T((f.Int64() * other.Int64()) >> D)
}

func (f T) Div(other T) T {
	return T((f.Int64() << D) / other.Int64())
}

func (f T) Int32() int32 {
	return int32(f)
}

func (f T) Int64() int64 {
	return int64(f)
}

func (f T) Int() int {
	return int(f)
}

func (f T) Float() float64 {
	return float64(f) / Denom
}

func (f T) Float32() float32 {
	return float32(f) / Denom
}

func FromFloat(f float64) T {
	return T(int32(f * Denom))
}

func FromFloat32(f float32) T {
	return T(int32(f * Denom))
}

func FromInt(i int) T {
	return T(i << D)
}
