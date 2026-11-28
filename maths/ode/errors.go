package ode

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.ode: "

var (
	// ErrNilFunction 表示未提供右端函数
	ErrNilFunction = errors.New(_ERROR_PREFIX + "function must not be nil")
	// ErrEmptyState 表示初始状态为空
	ErrEmptyState = errors.New(_ERROR_PREFIX + "initial state must not be empty")
	// ErrInvalidRange 表示积分区间非法
	ErrInvalidRange = errors.New(_ERROR_PREFIX + "range must satisfy start < stop with finite endpoints")
	// ErrInvalidStep 表示步长非正、非有限或导致步数超出上限
	ErrInvalidStep = errors.New(_ERROR_PREFIX + "step size must be positive and finite")
	// ErrInvalidTolerance 表示误差容差非正或非有限
	ErrInvalidTolerance = errors.New(_ERROR_PREFIX + "tolerance must be positive and finite")
	// ErrInvalidStepRange 表示步长上下界非法
	ErrInvalidStepRange = errors.New(_ERROR_PREFIX + "step range must satisfy 0 <= minimum <= maximum")
	// ErrInvalidIterations 表示最大步数非正
	ErrInvalidIterations = errors.New(_ERROR_PREFIX + "max iterations must be positive")
	// ErrTooManySteps 表示固定步长在区间内产生的步数超出上限
	ErrTooManySteps = errors.New(_ERROR_PREFIX + "step count exceeds the supported limit")
	// ErrStateLengthChanged 表示右端函数返回的分量数与状态维度不一致
	ErrStateLengthChanged = errors.New(_ERROR_PREFIX + "function returned a state of different length")
	// ErrNonFinite 表示计算过程中出现 NaN 或 Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
	// ErrNotConverged 表示达到步数上限仍未完成区间积分
	ErrNotConverged = errors.New(_ERROR_PREFIX + "failed to reach the end of the range within the iteration limit")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
