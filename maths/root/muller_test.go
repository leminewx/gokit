package root

import (
	"errors"
	"math"
	"math/cmplx"
	"testing"
)

func TestCalculateMullerComplexRoot(t *testing.T) {
	// z² + 1 = 0 的两个根为 ±i，需用复根求解
	solution, err := CalculateMuller(func(z complex128) complex128 { return z*z + 1 }, 0, 1, 2)
	if err != nil {
		t.Fatalf("CalculateMuller: %v", err)
	}

	if modulus := cmplx.Abs(solution.Root); math.Abs(modulus-1) > 1e-9 {
		t.Fatalf("|Root| = %.15g, want 1", modulus)
	}

	if math.Abs(real(solution.Root)) > 1e-9 {
		t.Fatalf("Re(Root) = %.15g, want 0", real(solution.Root))
	}

	if math.Abs(math.Abs(imag(solution.Root))-1) > 1e-9 {
		t.Fatalf("Im(Root) = %.15g, want ±1", imag(solution.Root))
	}

	if cmplx.Abs(solution.Value) > 1e-8 {
		t.Fatalf("Value = %v, want |f| <= 1e-8", solution.Value)
	}
}

func TestCalculateMullerRealRoot(t *testing.T) {
	// 实系数多项式同样适用：x³ - 2x - 5 = 0 的实根
	solution, err := CalculateMuller(func(z complex128) complex128 { return z*z*z - 2*z - 5 }, 2, 2.1, 2.2)
	if err != nil {
		t.Fatalf("CalculateMuller: %v", err)
	}

	requireClose(t, "Re(Root)", real(solution.Root), 2.0945514815423265, 1e-9)

	if cmplx.Abs(solution.Value) > 1e-8 {
		t.Fatalf("Value = %v, want |f| <= 1e-8", solution.Value)
	}
}

func TestCalculateMullerQuadratic(t *testing.T) {
	solution, err := CalculateMuller(func(z complex128) complex128 { return z*z - 2 }, 1+0i, 1.5+0i, 2+0i)
	if err != nil {
		t.Fatalf("CalculateMuller: %v", err)
	}

	requireClose(t, "|Root|", cmplx.Abs(solution.Root), math.Sqrt2, 1e-9)
}

func TestCalculateMullerRejectsDuplicateInitials(t *testing.T) {
	_, err := CalculateMuller(func(z complex128) complex128 { return z*z - 2 }, 1, 1, 2)
	if !errors.Is(err, ErrDuplicateInitial) {
		t.Fatalf("err = %v, want %v", err, ErrDuplicateInitial)
	}
}

func TestCalculateMullerNonFiniteInitial(t *testing.T) {
	_, err := CalculateMuller(func(z complex128) complex128 { return z }, complex(math.Inf(1), 0), 1, 2)
	if !errors.Is(err, ErrInvalidInitial) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidInitial)
	}
}

func TestCalculateMullerNonFinite(t *testing.T) {
	_, err := CalculateMuller(func(z complex128) complex128 { return complex(math.NaN(), 0) }, 0, 1, 2)
	if !errors.Is(err, ErrNonFinite) {
		t.Fatalf("err = %v, want %v", err, ErrNonFinite)
	}
}
