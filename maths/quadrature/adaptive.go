package quadrature

import "math"

// CalculateAdaptiveCotes 用自适应 Simpson 法计算定积分
//   - 递归细分：比较子区间的 Simpson 值与两个半区间 Simpson 值之和，以 |差值|/15 估计局部误差；
//   - 收敛判据：局部误差 <= tolerance * 子区间长度 / 总长度，即按长度比例分配容差；
//   - 递归深度上限由 WithMaxIterations 控制（默认 16），达到上限时接受当前结果而不报错；
//   - ErrorEstimate 为各叶子 |差值|/15 之和，始终填充；
//   - Iterations 为叶子区间个数，可据此判断细分规模。
func CalculateAdaptiveCotes(function Function, a, b float64, options ...Option) (Result, error) {
	if function == nil {
		return Result{}, ErrNilFunction
	}

	own := newConfig(options)
	if err := own.validate(); err != nil {
		return Result{}, err
	}

	if err := validateInterval(a, b); err != nil {
		return Result{}, err
	}

	state := &adaptiveState{
		function:  function,
		tolerance: own.tolerance,
		maxDepth:  own.maxIterations,
		width:     b - a,
	}

	left, middle, right := a, (a+b)/2, b
	valueLeft, valueMiddle, valueRight := function(left), function(middle), function(right)
	if !isFinite(valueLeft) || !isFinite(valueMiddle) || !isFinite(valueRight) {
		return Result{}, wrapError(ErrNonFinite, "function value at the initial nodes")
	}

	value, err := state.integrate(left, right, middle, valueLeft, valueMiddle, valueRight, 0)
	if err != nil {
		return Result{}, err
	}

	return Result{Value: value, Iterations: state.leaves, ErrorEstimate: state.estimate}, nil
}

// adaptiveState 保存自适应积分的共享状态
type adaptiveState struct {
	function  Function // 被积函数
	tolerance float64  // 全局容差
	maxDepth  int      // 递归深度上限
	width     float64  // 总区间长度，用于按比例分配容差
	leaves    int      // 已接受的叶子区间个数
	estimate  float64  // 累计误差指示
}

// integrate 递归细分 [left, right] 并返回该区间的积分近似
func (own *adaptiveState) integrate(left, right, middle, valueLeft, valueMiddle, valueRight float64, depth int) (float64, error) {
	leftMiddle := (left + middle) / 2
	rightMiddle := (middle + right) / 2
	valueLeftMiddle := own.function(leftMiddle)
	valueRightMiddle := own.function(rightMiddle)
	if !isFinite(valueLeftMiddle) || !isFinite(valueRightMiddle) {
		return 0, wrapError(ErrNonFinite, "subdivision of [%g, %g]", left, right)
	}

	whole := (right - left) / 6 * (valueLeft + 4*valueMiddle + valueRight)
	refinedLeft := (middle - left) / 6 * (valueLeft + 4*valueLeftMiddle + valueMiddle)
	refinedRight := (right - middle) / 6 * (valueMiddle + 4*valueRightMiddle + valueRight)
	refined := refinedLeft + refinedRight

	delta := refined - whole
	threshold := 15 * own.tolerance * (right - left) / own.width

	if depth >= own.maxDepth || math.Abs(delta) <= threshold {
		own.leaves++
		own.estimate += math.Abs(delta) / 15

		return refined + delta/15, nil
	}

	leftValue, err := own.integrate(left, middle, leftMiddle, valueLeft, valueLeftMiddle, valueMiddle, depth+1)
	if err != nil {
		return 0, err
	}

	rightValue, err := own.integrate(middle, right, rightMiddle, valueMiddle, valueRightMiddle, valueRight, depth+1)
	if err != nil {
		return 0, err
	}

	return leftValue + rightValue, nil
}
