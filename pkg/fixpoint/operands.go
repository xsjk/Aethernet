package fixpoint

type Fixpoint int32

func NewFixpoint(value int32) Fixpoint {
	return Fixpoint(value)
}

func (f Fixpoint) Int32() int32 {
	return int32(f)
}

func (f Fixpoint) Add(other Fixpoint) Fixpoint {
	return Fixpoint(f.Int32() + other.Int32())
}

func (f Fixpoint) Sub(other Fixpoint) Fixpoint {
	return Fixpoint(f.Int32() - other.Int32())
}

func (f Fixpoint) Mul(other Fixpoint) Fixpoint {
	return Fixpoint((f.Int32() * other.Int32()) >> 15)
}

func (f Fixpoint) Div(other Fixpoint) Fixpoint {
	return Fixpoint((f.Int32() << 15) / other.Int32())
}

func (f Fixpoint) MulInt32(other int32) Fixpoint {
	return Fixpoint(f.Int32() * other)
}

func (f Fixpoint) DivInt32(other int32) Fixpoint {
	return Fixpoint(f.Int32() / other)
}

func (f Fixpoint) Abs() Fixpoint {
	if f.Int32() < 0 {
		return -f
	}
	return f
}

func (f Fixpoint) ToFloat() float64 {
	return float64(f.Int32()) / 32768.0
}

func (f Fixpoint) ToFloat32() float32 {
	return float32(f.Int32()) / 32768.0
}

func FromFloat(f float64) Fixpoint {
	return Fixpoint(int32(f * 32768.0))
}

func FromFloat32(f float32) Fixpoint {
	return Fixpoint(int32(f * 32768.0))
}
