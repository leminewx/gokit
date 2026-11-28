package knn

import (
	"errors"
	"fmt"
)

const _ERROR_PREFIX = "gokit.maths.learning.knn: "

var (
	// ErrNoSamples 表示训练样本为空
	ErrNoSamples = errors.New(_ERROR_PREFIX + "no samples provided")
	// ErrLabelMismatch 表示样本数量与标签数量不一致
	ErrLabelMismatch = errors.New(_ERROR_PREFIX + "number of samples and labels mismatch")
	// ErrNoFeatures 表示样本不包含任何特征
	ErrNoFeatures = errors.New(_ERROR_PREFIX + "samples must have at least one feature")
	// ErrDimensionMismatch 表示样本或预测输入的特征维度不一致
	ErrDimensionMismatch = errors.New(_ERROR_PREFIX + "feature dimensions mismatch")
	// ErrNotTrained 表示模型尚未训练
	ErrNotTrained = errors.New(_ERROR_PREFIX + "model is not trained")
	// ErrInvalidK 表示近邻数量非法（必须为正）
	ErrInvalidK = errors.New(_ERROR_PREFIX + "k must be positive")
)

// wrapDimensionError 构造维度不一致的错误，保留可用于 errors.Is 的哨兵错误
func wrapDimensionError(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrDimensionMismatch}, args...)...)
}
