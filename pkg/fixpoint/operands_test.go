package fixpoint

import (
	"testing"
)

func TestNewFixpoint(t *testing.T) {
	value := int32(12345)
	fp := NewFixpoint(value)
	if fp.Int32() != value {
		t.Errorf("Expected %d, got %d", value, fp.Int32())
	}
}

func TestFixpoint_Add(t *testing.T) {
	a := NewFixpoint(10000)
	b := NewFixpoint(20000)
	expected := NewFixpoint(30000)
	result := a.Add(b)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFixpoint_Sub(t *testing.T) {
	a := NewFixpoint(20000)
	b := NewFixpoint(10000)
	expected := NewFixpoint(10000)
	result := a.Sub(b)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFixpoint_Mul(t *testing.T) {
	a := NewFixpoint(16384)       // 0.5 in Q15
	b := NewFixpoint(16384)       // 0.5 in Q15
	expected := NewFixpoint(8192) // 0.25 in Q15
	result := a.Mul(b)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFixpoint_Div(t *testing.T) {
	a := NewFixpoint(16384)        // 0.5 in Q15
	b := NewFixpoint(32768)        // 1.0 in Q15
	expected := NewFixpoint(16384) // 0.5 in Q15
	result := a.Div(b)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFixpoint_MulInt32(t *testing.T) {
	a := NewFixpoint(10000)
	b := int32(2)
	expected := NewFixpoint(20000)
	result := a.MulInt32(b)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFixpoint_DivInt32(t *testing.T) {
	a := NewFixpoint(20000)
	b := int32(2)
	expected := NewFixpoint(10000)
	result := a.DivInt32(b)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFixpoint_Abs(t *testing.T) {
	a := NewFixpoint(-10000)
	expected := NewFixpoint(10000)
	result := a.Abs()
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFixpoint_ToFloat(t *testing.T) {
	a := NewFixpoint(16384) // 0.5 in Q15
	expected := 0.5
	result := a.ToFloat()
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestFixpoint_ToFloat32(t *testing.T) {
	a := NewFixpoint(16384) // 0.5 in Q15
	expected := float32(0.5)
	result := a.ToFloat32()
	if result != expected {
		t.Errorf("Expected %f, got %f", expected, result)
	}
}

func TestFromFloat(t *testing.T) {
	f := 0.5
	expected := NewFixpoint(16384) // 0.5 in Q15
	result := FromFloat(f)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

func TestFromFloat32(t *testing.T) {
	f := float32(0.5)
	expected := NewFixpoint(16384) // 0.5 in Q15
	result := FromFloat32(f)
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}
