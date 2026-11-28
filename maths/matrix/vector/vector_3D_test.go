package vector

import (
	"errors"
	"math"
	"testing"
)

func TestVector3D(t *testing.T) {
	t.Parallel()

	vec := Vector3D[float64]{1, 2, 2}
	other := Vector3D[float64]{2, 1, 0}

	dot, err := vec.CalculateDot(&other)
	if err != nil || dot != 4 {
		t.Errorf("Dot = (%v, %v), want (4, nil)", dot, err)
	}

	sum, err := vec.Add(&other)
	if err != nil {
		t.Fatalf("Add 返回错误: %v", err)
	}
	if got := sum.(*Vector3D[float64]); got[0] != 3 || got[1] != 3 || got[2] != 2 {
		t.Errorf("Add = %v, want [3 3 2]", *got)
	}

	diff, err := vec.Subtract(&other)
	if err != nil {
		t.Fatalf("Subtract 返回错误: %v", err)
	}
	if got := diff.(*Vector3D[float64]); got[0] != -1 || got[1] != 1 || got[2] != 2 {
		t.Errorf("Subtract = %v, want [-1 1 2]", *got)
	}

	if got := vec.GetNorm(); math.Abs(got-3) > 1e-9 {
		t.Errorf("GetNorm() = %v, want 3", got)
	}
	if shape := vec.GetShape(); len(shape) != 1 || shape[0] != 3 {
		t.Errorf("GetShape() = %v, want [3]", shape)
	}
}

func TestVector3DCross(t *testing.T) {
	t.Parallel()

	// x 轴叉乘 y 轴应为 z 轴
	xAxis := Vector3D[int]{1, 0, 0}
	yAxis := Vector3D[int]{0, 1, 0}

	cross, err := xAxis.CalculateCross(&yAxis)
	if err != nil {
		t.Fatalf("Cross 返回错误: %v", err)
	}
	got, ok := cross.(*Vector3D[int])
	if !ok {
		t.Fatalf("Cross 返回类型 = %T, want *Vector3D[int]", cross)
	}
	if got[0] != 0 || got[1] != 0 || got[2] != 1 {
		t.Errorf("Cross = %v, want [0 0 1]", *got)
	}

	// 平行向量叉积为零向量
	parallel, err := xAxis.CalculateCross(&Vector3D[int]{2, 0, 0})
	if err != nil {
		t.Fatalf("Cross 返回错误: %v", err)
	}
	if v := parallel.(*Vector3D[int]); v[0] != 0 || v[1] != 0 || v[2] != 0 {
		t.Errorf("平行向量 Cross = %v, want [0 0 0]", *v)
	}
}

func TestVector3DIndex(t *testing.T) {
	t.Parallel()

	vec := Vector3D[int]{1, 2, 3}
	if err := vec.SetValueAt(9, 2); err != nil || vec[2] != 9 {
		t.Errorf("SetValueAt(9, 2) = %v, vec = %v", err, vec)
	}

	for _, index := range []int{-1, 3} {
		if err := vec.SetValueAt(1, index); !errors.Is(err, ErrIndexOutOfRange) {
			t.Errorf("SetValueAt(1, %d) 错误 = %v, want ErrIndexOutOfRange", index, err)
		}
		if _, err := vec.GetValueAt(index); !errors.Is(err, ErrIndexOutOfRange) {
			t.Errorf("GetValueAt(%d) 错误 = %v, want ErrIndexOutOfRange", index, err)
		}
	}
	if err := vec.SetValueAt(1); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("SetValueAt() 错误 = %v, want ErrIndexOutOfRange", err)
	}
	if _, err := vec.GetValueAt(); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("GetValueAt() 错误 = %v, want ErrIndexOutOfRange", err)
	}
}

func TestVector3DShapeMismatch(t *testing.T) {
	t.Parallel()

	vec := Vector3D[int]{1, 2, 3}
	other := Vector2D[int]{1, 2}

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
}
