package vector

import (
	"math"

	"gitee.com/leminewx/gokit"
)

type Vector2D[T gokit.Number] [2]T

var _ Vector[int] = (*Vector2D[int])(nil)

// CalculateDot 向量点积
func (own *Vector2D[T]) CalculateDot(vec Vector[T]) (T, error) {
	if !own.IsShapeEqualTo(vec) {
		return *new(T), ErrShapeNotMatch
	}

	var value T
	for i := range 2 {
		val, _ := vec.GetValueAt(i)
		value += own[i] * val
	}

	return value, nil
}

// Add 向量加法
func (own *Vector2D[T]) Add(vec Vector[T]) (Vector[T], error) {
	if !own.IsShapeEqualTo(vec) {
		return nil, ErrShapeNotMatch
	}

	var value Vector2D[T]
	for i := range 2 {
		val, _ := vec.GetValueAt(i)
		value[i] = own[i] + val
	}

	return &value, nil
}

// CalculateCross 向量叉积
func (own *Vector2D[T]) CalculateCross(vec Vector[T]) (Vector[T], error) {
	if !own.IsShapeEqualTo(vec) {
		return nil, ErrShapeNotMatch
	}

	val0, _ := vec.GetValueAt(0)
	val1, _ := vec.GetValueAt(1)
	return &Vector1D[T]{own[0]*val1 - own[1]*val0}, nil
}

// Subtract 向量减法
func (own *Vector2D[T]) Subtract(vec Vector[T]) (Vector[T], error) {
	if !own.IsShapeEqualTo(vec) {
		return nil, ErrShapeNotMatch
	}

	var value Vector2D[T]
	for i := range 2 {
		val, _ := vec.GetValueAt(i)
		value[i] = own[i] - val
	}

	return &value, nil
}

// SetValueAt 设置指定索引的元素值
func (own *Vector2D[T]) SetValueAt(value T, index ...int) error {
	if len(index) != 1 {
		return ErrIndexOutOfRange
	}

	switch index[0] {
	case 0:
		own[0] = value
		return nil
	case 1:
		own[1] = value
		return nil
	default:
		return ErrIndexOutOfRange
	}
}

// GetNorm 获取向量的模
func (own *Vector2D[T]) GetNorm() float64 {
	return math.Sqrt(float64(own[0]*own[0] + own[1]*own[1]))
}

// GetShape 获取向量的形状
func (own *Vector2D[T]) GetShape() []int {
	return []int{2}
}

// GetValueAt 获取指定索引的元素值
func (own *Vector2D[T]) GetValueAt(index ...int) (T, error) {
	if len(index) != 1 || index[0] < 0 || 1 < index[0] {
		return *new(T), ErrIndexOutOfRange
	}

	return own[index[0]], nil
}

// IsShapeEqualTo 判断两个向量是否具有相同的形状
func (own *Vector2D[T]) IsShapeEqualTo(vec Vector[T]) bool {
	shape := vec.GetShape()
	return len(shape) == 1 && shape[0] == 2
}
