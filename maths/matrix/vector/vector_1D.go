package vector

import (
	"math"

	"gitee.com/leminewx/gokit"
)

type Vector1D[T gokit.Number] [1]T

var _ Vector[int] = &Vector1D[int]{}

// CalculateDot 向量点积
func (own *Vector1D[T]) CalculateDot(vec Vector[T]) (T, error) {
	if !own.IsShapeEqualTo(vec) {
		return *new(T), ErrShapeNotMatch
	}

	val, _ := vec.GetValueAt(0)
	return own[0] * val, nil
}

// Add 向量加法
func (own *Vector1D[T]) Add(vec Vector[T]) (Vector[T], error) {
	if !own.IsShapeEqualTo(vec) {
		return nil, ErrShapeNotMatch
	}

	val, _ := vec.GetValueAt(0)
	return &Vector1D[T]{own[0] + val}, nil
}

// CalculateCross 向量叉积
func (own *Vector1D[T]) CalculateCross(vec Vector[T]) (Vector[T], error) {
	return nil, ErrOneDimensionalDotMeaningless
}

// Subtract 向量减法
func (own *Vector1D[T]) Subtract(vec Vector[T]) (Vector[T], error) {
	if !own.IsShapeEqualTo(vec) {
		return nil, ErrShapeNotMatch
	}

	val, _ := vec.GetValueAt(0)
	return &Vector1D[T]{own[0] - val}, nil
}

// SetValueAt 设置指定索引的元素值
func (own *Vector1D[T]) SetValueAt(value T, index ...int) error {
	if len(index) != 1 || index[0] != 0 {
		return ErrIndexOutOfRange
	}

	own[0] = value
	return nil
}

// GetNorm 获取向量的模
func (own *Vector1D[T]) GetNorm() float64 {
	return math.Abs(float64(own[0]))
}

// GetShape 获取向量的形状
func (own *Vector1D[T]) GetShape() []int {
	return []int{1}
}

// GetValueAt 获取指定索引的元素值
func (own *Vector1D[T]) GetValueAt(index ...int) (T, error) {
	if len(index) != 1 || index[0] != 0 {
		return *new(T), ErrIndexOutOfRange
	}

	return own[0], nil
}

// IsShapeEqualTo 判断两个向量是否具有相同的形状
func (own *Vector1D[T]) IsShapeEqualTo(vec Vector[T]) bool {
	shape := vec.GetShape()
	return len(shape) == 1 && shape[0] == 1
}
