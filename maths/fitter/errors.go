package fitter

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.fitter: "

var (
	// ErrEmptyInput 表示未提供样本
	ErrEmptyInput = errors.New(_ERROR_PREFIX + "samples must not be empty")
	// ErrInvalidDegree 表示多项式阶数超出支持范围
	ErrInvalidDegree = errors.New(_ERROR_PREFIX + "degree must be within the supported range")
	// ErrTooFewSamples 表示有效样本数不足以确定系数
	ErrTooFewSamples = errors.New(_ERROR_PREFIX + "not enough weighted samples for the requested degree")
	// ErrNegativeWeight 表示样本权重为负
	ErrNegativeWeight = errors.New(_ERROR_PREFIX + "weight must not be negative")
	// ErrNonFinite 表示样本或中间结果出现 NaN / Inf
	ErrNonFinite = errors.New(_ERROR_PREFIX + "encountered a non-finite value")
	// ErrSingularSystem 表示设计矩阵列线性相关，无法唯一确定系数
	ErrSingularSystem = errors.New(_ERROR_PREFIX + "design matrix is singular")
)

// wrapError 构造携带哨兵错误的包装错误，可用 errors.Is 判定
func wrapError(sentinel error, format string, args ...any) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: %s", sentinel, format)
	}
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}
