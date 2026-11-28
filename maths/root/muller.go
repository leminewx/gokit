package root

import (
	"math/cmplx"
)

// CalculateMuller 用 Muller 抛物线法求复根
//   - 需要三个互不相同的初值，重复时返回 ErrDuplicateInitial；
//   - 每次迭代用三点抛物线插值构造新近似，可同时得到实根与复根；
//   - 收敛判据：|Δz| <= tolerance，或启用残差容差且 |f(z)| <= residual；
//   - 适用：实系数多项式的复根（如 z²+1 = 0）、特征方程求根。
func CalculateMuller(function ComplexFunction, z0, z1, z2 complex128, options ...Option) (ComplexSolution, error) {
	if function == nil {
		return ComplexSolution{}, ErrNilFunction
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return ComplexSolution{}, err
	}

	if !isFiniteComplex(z0) || !isFiniteComplex(z1) || !isFiniteComplex(z2) {
		return ComplexSolution{}, wrapError(ErrInvalidInitial, "%v, %v, %v", z0, z1, z2)
	}

	if z0 == z1 || z1 == z2 || z0 == z2 {
		return ComplexSolution{}, wrapError(ErrDuplicateInitial, "%v, %v, %v", z0, z1, z2)
	}

	value0, value1, value2 := function(z0), function(z1), function(z2)
	if !isFiniteComplex(value0) || !isFiniteComplex(value1) || !isFiniteComplex(value2) {
		return ComplexSolution{}, wrapError(ErrNonFinite, "function value at the initial points")
	}

	for iteration := 1; iteration <= own.maxIterations; iteration++ {
		step1 := z1 - z0
		step2 := z2 - z1

		slope1 := (value1 - value0) / step1
		slope2 := (value2 - value1) / step2

		span := step1 + step2
		if cmplx.Abs(span) < _ZERO_THRESHOLD {
			return ComplexSolution{}, wrapError(ErrZeroDerivative, "adjacent steps collapsed at z=%v", z2)
		}

		curvature := (slope2 - slope1) / span
		linear := slope2 + step2*curvature

		// 选取模更大的分母，避免相消误差（Muller 法的标准做法）
		discriminant := cmplx.Sqrt(linear*linear - 4*curvature*value2)
		divisor := linear + discriminant
		if cmplx.Abs(linear-discriminant) > cmplx.Abs(divisor) {
			divisor = linear - discriminant
		}

		if cmplx.Abs(divisor) < _ZERO_THRESHOLD {
			return ComplexSolution{}, wrapError(ErrZeroDerivative, "parabolic divisor vanished at z=%v", z2)
		}

		next := z2 - 2*value2/divisor
		if !isFiniteComplex(next) {
			return ComplexSolution{}, wrapError(ErrNonFinite, "muller step at z=%v", z2)
		}

		value := function(next)
		if !isFiniteComplex(value) {
			return ComplexSolution{}, wrapError(ErrNonFinite, "f(%v)", next)
		}

		if own.isConverged(cmplx.Abs(next-z2), cmplx.Abs(value)) {
			return ComplexSolution{Root: next, Value: value, Iterations: iteration}, nil
		}

		z0, z1, z2 = z1, z2, next
		value0, value1, value2 = value1, value2, value
	}

	return ComplexSolution{Root: z2, Value: value2, Iterations: own.maxIterations},
		wrapError(ErrNotConverged, "%d iterations, last z=%v", own.maxIterations, z2)
}
