package vector

import (
	"gitee.com/leminewx/gokit"
)

// Vector 向量接口
type Vector[T gokit.Number] interface {
	// CalculateDot 向量点积
	CalculateDot(Vector[T]) (T, error)
	// Add 向量加法
	Add(Vector[T]) (Vector[T], error)
	// CalculateCross 向量叉积
	CalculateCross(Vector[T]) (Vector[T], error)
	// Subtract 向量减法
	Subtract(Vector[T]) (Vector[T], error)
	// SetValueAt 设置指定索引的元素值
	SetValueAt(value T, index ...int) error

	// GetNorm 获取向量的模
	GetNorm() float64
	// GetShape 获取向量的形状
	GetShape() []int
	// GetValueAt 获取指定索引的元素值
	GetValueAt(index ...int) (T, error)
	// IsShapeEqualTo 判断两个向量形状是否相同
	IsShapeEqualTo(Vector[T]) bool
}

// 编译期断言：一维、二维、三维向量均实现 Vector 接口
var (
	_ Vector[int]     = (*Vector1D[int])(nil)
	_ Vector[float64] = (*Vector2D[float64])(nil)
	_ Vector[float64] = (*Vector3D[float64])(nil)
)

// CalculateWedge 外积（反对称矩阵）
func CalculateWedge[T gokit.Number](a, b []T) ([][]T, bool) {
	lenA, lenB := len(a), len(b)
	if lenA == 0 || lenA != lenB {
		return [][]T{}, false
	}

	mat := make([][]T, lenA)
	for i := 0; i < lenA; i++ {
		mat[i] = make([]T, lenA)
		for j := 0; j < lenA; j++ {
			mat[i][j] = a[i]*b[j] - a[j]*b[i]
		}
	}

	return mat, true
}
