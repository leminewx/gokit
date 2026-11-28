package vector

import (
	"errors"
	"math"
	"testing"
)

func TestVector2D(t *testing.T) {
	t.Parallel()

	vec := Vector2D[float64]{3, 4}
	other := Vector2D[float64]{1, 2}

	dot, err := vec.CalculateDot(&other)
	if err != nil || dot != 11 {
		t.Errorf("Dot = (%v, %v), want (11, nil)", dot, err)
	}

	sum, err := vec.Add(&other)
	if err != nil {
		t.Fatalf("Add 返回错误: %v", err)
	}
	if got := sum.(*Vector2D[float64]); got[0] != 4 || got[1] != 6 {
		t.Errorf("Add = %v, want [4 6]", *got)
	}

	diff, err := vec.Subtract(&other)
	if err != nil {
		t.Fatalf("Subtract 返回错误: %v", err)
	}
	if got := diff.(*Vector2D[float64]); got[0] != 2 || got[1] != 2 {
		t.Errorf("Subtract = %v, want [2 2]", *got)
	}

	if got := vec.GetNorm(); math.Abs(got-5) > 1e-9 {
		t.Errorf("GetNorm() = %v, want 5", got)
	}
	if shape := vec.GetShape(); len(shape) != 1 || shape[0] != 2 {
		t.Errorf("GetShape() = %v, want [2]", shape)
	}
}

func TestVector2DCross(t *testing.T) {
	t.Parallel()

	vec := Vector2D[int]{3, 4}
	other := Vector2D[int]{1, 2}

	cross, err := vec.CalculateCross(&other)
	if err != nil {
		t.Fatalf("Cross 返回错误: %v", err)
	}

	// 二维叉积为标量：x1*y2 - y1*x2
	if got, ok := cross.(*Vector1D[int]); !ok || got[0] != 2 {
		t.Errorf("Cross = %v, want 2", cross)
	}
}

func TestVector2DIndex(t *testing.T) {
	t.Parallel()

	vec := Vector2D[int]{1, 2}
	if err := vec.SetValueAt(9, 1); err != nil || vec[1] != 9 {
		t.Errorf("SetValueAt(9, 1) = %v, vec = %v", err, vec)
	}

	if err := vec.SetValueAt(1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("SetValueAt() 错误 = %v, want ErrIndexOutOfRange", err)
	}
	if err := vec.SetValueAt(1, 2); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("SetValueAt(1, 2) 错误 = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := vec.GetValueAt(-1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("GetValueAt(-1) 错误 = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := vec.GetValueAt(); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("GetValueAt() 错误 = %v, want ErrIndexOutOfRange", err)
	}
	if got, err := vec.GetValueAt(0); err != nil || got != 1 {
		t.Errorf("GetValueAt(0) = (%v, %v), want (1, nil)", got, err)
	}
}

func TestVector2DShapeMismatch(t *testing.T) {
	t.Parallel()

	vec := Vector2D[int]{1, 2}
	other := Vector3D[int]{1, 2, 3}

	if _, err := vec.CalculateDot(&other); !errors.Is(err, ErrShapeNotMatch) {
		t.Errorf("Dot 错误 = %v, want ErrShapeNotMatch", err)
	}
	if _, err := vec.Add(&other); !errors.Is(err, ErrShapeNotMatch) {
		t.Errorf("Add 错误 = %v, want ErrShapeNotMatch", err)
	}
	if _, err := vec.CalculateCross(&other); !errors.Is(err, ErrShapeNotMatch) {
		t.Errorf("Cross 错误 = %v, want ErrShapeNotMatch", err)
	}
	if _, err := vec.Subtract(&other); !errors.Is(err, ErrShapeNotMatch) {
		t.Errorf("Subtract 错误 = %v, want ErrShapeNotMatch", err)
	}
	if vec.IsShapeEqualTo(&other) {
		t.Error("IsShapeEqualTo(3D) = true, want false")
	}
}
