package interpolator

import (
	"errors"
	"math"
	"testing"
)

// calculateResidual 计算三对角矩阵与解的乘积，用于按原方程校验求解结果
func calculateResidual(lower, diagonal, upper, solution []float64) []float64 {
	residual := make([]float64, len(solution))
	for index := range solution {
		residual[index] = diagonal[index] * solution[index]
		if index > 0 {
			residual[index] += lower[index] * solution[index-1]
		}
		if index < len(solution)-1 {
			residual[index] += upper[index] * solution[index+1]
		}
	}

	return residual
}

func TestSolveTridiagonal(t *testing.T) {
	tests := []struct {
		name     string
		lower    []float64
		diagonal []float64
		upper    []float64
		right    []float64
		want     []float64
	}{
		{
			name:     "3x3 严格对角占优",
			lower:    []float64{0, 1, 1},
			diagonal: []float64{2, 3, 2},
			upper:    []float64{1, 1, 0},
			right:    []float64{1, 2, 3},
			want:     []float64{0.5, 0, 1.5},
		},
		{
			name:     "4x4 单位矩阵",
			lower:    []float64{0, 0, 0, 0},
			diagonal: []float64{1, 1, 1, 1},
			upper:    []float64{0, 0, 0, 0},
			right:    []float64{1, 2, 3, 4},
			want:     []float64{1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := solveTridiagonal(tt.lower, tt.diagonal, tt.upper, tt.right)
			if err != nil {
				t.Fatalf("solveTridiagonal() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("solveTridiagonal() 长度 = %d, want %d", len(got), len(tt.want))
			}

			for index := range tt.want {
				if math.Abs(got[index]-tt.want[index]) > 1e-12 {
					t.Fatalf("解[%d] = %g, want %g", index, got[index], tt.want[index])
				}
			}

			residual := calculateResidual(tt.lower, tt.diagonal, tt.upper, got)
			for index := range residual {
				if math.Abs(residual[index]-tt.right[index]) > 1e-12 {
					t.Fatalf("残差[%d] = %g, want %g", index, residual[index], tt.right[index])
				}
			}
		})
	}
}

func TestSolveTridiagonalInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		lower    []float64
		diagonal []float64
		upper    []float64
		right    []float64
	}{
		{"空方程组", nil, nil, nil, nil},
		{"首行主元为零", []float64{0, 0}, []float64{0, 1}, []float64{1, 0}, []float64{1, 1}},
		{"消元后主元为零", []float64{0, 1}, []float64{1, 1}, []float64{1, 0}, []float64{1, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := solveTridiagonal(tt.lower, tt.diagonal, tt.upper, tt.right); !errors.Is(err, ErrSingularSystem) {
				t.Fatalf("solveTridiagonal() error = %v, want %v", err, ErrSingularSystem)
			}
		})
	}
}

func TestSolveCyclicTridiagonal(t *testing.T) {
	// 循环三对角：[[4,1,1],[1,4,1],[1,1,4]] * [1,1,1] = [6,6,6]
	lower := []float64{1, 1, 1}
	diagonal := []float64{4, 4, 4}
	upper := []float64{1, 1, 1}
	right := []float64{6, 6, 6}

	got, err := solveCyclicTridiagonal(lower, diagonal, upper, right, upper[len(upper)-1], lower[0])
	if err != nil {
		t.Fatalf("solveCyclicTridiagonal() error = %v", err)
	}

	for index := range got {
		if math.Abs(got[index]-1) > 1e-12 {
			t.Fatalf("解[%d] = %g, want 1", index, got[index])
		}
	}
}

func TestSolveCyclicTridiagonalInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		diagonal []float64
	}{
		{"未知量不足", []float64{4, 4}},
		{"循环主元为零", []float64{0, 4, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := len(tt.diagonal)
			lower := make([]float64, size)
			upper := make([]float64, size)
			right := make([]float64, size)

			_, err := solveCyclicTridiagonal(lower, tt.diagonal, upper, right, 1, 1)
			if !errors.Is(err, ErrSingularSystem) {
				t.Fatalf("solveCyclicTridiagonal() error = %v, want %v", err, ErrSingularSystem)
			}
		})
	}
}

func TestCalculatePeriodicMoments(t *testing.T) {
	ys := []float64{0, 1, 0, 0}
	steps := []float64{1, 1, 1}

	moments, err := calculatePeriodicMoments(ys, steps)
	if err != nil {
		t.Fatalf("calculatePeriodicMoments() error = %v", err)
	}
	if len(moments) != len(ys) {
		t.Fatalf("二阶导数长度 = %d, want %d", len(moments), len(ys))
	}
	if moments[len(moments)-1] != moments[0] {
		t.Fatalf("周期端点二阶导数 = %g, want %g", moments[len(moments)-1], moments[0])
	}

	for index, value := range moments {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("二阶导数[%d] = %g, want 有限值", index, value)
		}
	}
}

func TestValidateBoundary(t *testing.T) {
	clamped := &config{hasClampedBoundary: true, leftDerivative: 1, rightDerivative: 1}

	tests := []struct {
		name      string
		condition BoundaryCondition
		xs        []float64
		ys        []float64
		cfg       *config
		wantErr   error
	}{
		{"自然边界", BOUNDARY_NATURAL, []float64{0, 1, 2}, []float64{0, 1, 2}, &config{}, nil},
		{"未知外推策略", BOUNDARY_NATURAL, []float64{0, 1, 2}, []float64{0, 1, 2}, &config{policy: EXTRAPOLATION_WRAP + 1}, ErrInvalidBoundary},
		{"未知边界条件", BoundaryCondition(200), []float64{0, 1, 2}, []float64{0, 1, 2}, &config{}, ErrInvalidBoundary},
		{"夹持缺导数", BOUNDARY_CLAMPED, []float64{0, 1, 2}, []float64{0, 1, 2}, &config{}, ErrMissingDerivative},
		{"夹持导数非有限", BOUNDARY_CLAMPED, []float64{0, 1, 2}, []float64{0, 1, 2}, &config{hasClampedBoundary: true, leftDerivative: math.Inf(1)}, ErrNonFinite},
		{"夹持边界", BOUNDARY_CLAMPED, []float64{0, 1, 2}, []float64{0, 1, 2}, clamped, nil},
		{"周期采样不足", BOUNDARY_PERIODIC, []float64{0, 1, 2}, []float64{1, 2, 1}, &config{}, ErrInvalidBoundary},
		{"周期端点值不等", BOUNDARY_PERIODIC, []float64{0, 1, 2, 3}, []float64{1, 2, 3, 4}, &config{}, ErrInvalidBoundary},
		{"周期边界", BOUNDARY_PERIODIC, []float64{0, 1, 2, 3}, []float64{1, 2, 3, 1}, &config{}, nil},
		{"非扭结采样不足", BOUNDARY_NOT_A_KNOT, []float64{0, 1, 2}, []float64{0, 1, 2}, &config{}, ErrInvalidBoundary},
		{"非扭结边界", BOUNDARY_NOT_A_KNOT, []float64{0, 1, 2, 3}, []float64{0, 1, 4, 9}, &config{}, nil},
		{"周期环绕需周期边界", BOUNDARY_NATURAL, []float64{0, 1, 2, 3}, []float64{0, 1, 4, 9}, &config{policy: EXTRAPOLATION_WRAP}, ErrInvalidBoundary},
		{"周期环绕", BOUNDARY_PERIODIC, []float64{0, 1, 2, 3}, []float64{1, 2, 3, 1}, &config{policy: EXTRAPOLATION_WRAP}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBoundary(tt.condition, tt.xs, tt.ys, tt.cfg)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validateBoundary() error = %v, want nil", err)
				}

				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateBoundary() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
