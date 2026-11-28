package vector

import (
	"errors"
	"math"
	"testing"
)

func TestVector1D(t *testing.T) {
	t.Parallel()

	vec := Vector1D[float64]{3}
	other := Vector1D[float64]{2}

	dot, err := vec.CalculateDot(&other)
	if err != nil || dot != 6 {
		t.Errorf("Dot = (%v, %v), want (6, nil)", dot, err)
	}

	sum, err := vec.Add(&other)
	if err != nil {
		t.Fatalf("Add 返回错误: %v", err)
	}
	if sumVal, _ := sum.GetValueAt(0); sumVal != 5 {
		t.Errorf("Add = %v, want 5", sumVal)
	}

	diff, err := vec.Subtract(&other)
	if err != nil {
		t.Fatalf("Subtract 返回错误: %v", err)
	}
	if diffVal, _ := diff.GetValueAt(0); diffVal != 1 {
		t.Errorf("Subtract = %v, want 1", diffVal)
	}

	if got := vec.GetNorm(); got != 3 {
		t.Errorf("GetNorm() = %v, want 3", got)
	}
	if shape := vec.GetShape(); len(shape) != 1 || shape[0] != 1 {
		t.Errorf("GetShape() = %v, want [1]", shape)
	}
}

func TestVector1DIndex(t *testing.T) {
	t.Parallel()

	vec := Vector1D[int]{1}
	if err := vec.SetValueAt(9, 0); err != nil || vec[0] != 9 {
		t.Errorf("SetValueAt(9, 0) = %v, vec = %v", err, vec)
	}

	// 缺少索引或索引越界必须返回错误而不是 panic
	if err := vec.SetValueAt(1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("SetValueAt() 错误 = %v, want ErrIndexOutOfRange", err)
	}
	if err := vec.SetValueAt(1, 1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("SetValueAt(1, 1) 错误 = %v, want ErrIndexOutOfRange", err)
	}

	if _, err := vec.GetValueAt(); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("GetValueAt() 错误 = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := vec.GetValueAt(1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("GetValueAt(1) 错误 = %v, want ErrIndexOutOfRange", err)
	}
}

func TestVector1DShapeMismatch(t *testing.T) {
	t.Parallel()

	vec := Vector1D[int]{1}
	other := Vector2D[int]{1, 2}

	if _, err := vec.CalculateDot(&other); !errors.Is(err, ErrShapeNotMatch) {
		t.Errorf("Dot 错误 = %v, want ErrShapeNotMatch", err)
	}
	if _, err := vec.Add(&other); !errors.Is(err, ErrShapeNotMatch) {
		t.Errorf("Add 错误 = %v, want ErrShapeNotMatch", err)
	}
	if _, err := vec.Subtract(&other); !errors.Is(err, ErrShapeNotMatch) {
		t.Errorf("Subtract 错误 = %v, want ErrShapeNotMatch", err)
	}
	if _, err := vec.CalculateCross(&other); !errors.Is(err, ErrOneDimensionalDotMeaningless) {
		t.Errorf("Cross 错误 = %v, want ErrOneDimensionalDotMeaningless", err)
	}
	if vec.IsShapeEqualTo(&other) {
		t.Error("IsShapeEqualTo(2D) = true, want false")
	}
}

func TestVector1DNormNegative(t *testing.T) {
	t.Parallel()

	vec := Vector1D[float64]{-2.5}
	if got := vec.GetNorm(); math.Abs(got-2.5) > 1e-9 {
		t.Errorf("GetNorm() = %v, want 2.5", got)
	}
}
