package learning

import (
	"math"

	"gitee.com/leminewx/gokit"
)

// DistanceCalculator 距离计算器
type DistanceCalculator[T gokit.Number] func(a, b []T) float64

// CalculateEuclideanDistance 计算 a、b 两点的欧式距离
func CalculateEuclideanDistance[T gokit.Number](a, b []T) float64 {
	if len(a) != len(b) {
		return math.Inf(1)
	}

	sum := 0.0
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += float64(diff) * float64(diff)
	}

	return math.Sqrt(sum)
}
