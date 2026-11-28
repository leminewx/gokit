package maths

import (
	"math"
	"sort"

	"gitee.com/leminewx/gokit"
)

// CalculateStandardDeviation 计算一组数据的标准差
//   - mean: 给定均值，可选（内部会自动计算均值）
//   - 若输入为空切片，返回 0.0
func CalculateStandardDeviation[T gokit.Number](values []T, mean ...float64) float64 {
	return math.Sqrt(CalculateVariance(values, mean...))
}

// CalculateVariance 计算一组数据的总体方差
//   - mean: 给定均值，可选（内部会自动计算均值）
//   - 若输入为空切片，返回 0.0
func CalculateVariance[T gokit.Number](values []T, mean ...float64) float64 {
	n := len(values)
	if n == 0 {
		return 0.0
	}

	// 1. 计算均值
	var meanValue float64
	if len(mean) > 0 {
		meanValue = mean[0]
	} else {
		meanValue = CalculateMean(values...)
	}

	// 2. 计算离差平方和
	var sqrtSum float64
	for _, val := range values {
		diff := float64(val) - meanValue
		sqrtSum += diff * diff // 等价于 math.Pow(diff, 2)
	}

	// 3. 总体方差 = 平方和 / n
	return sqrtSum / float64(n)
}

// CalculateMedian 计算一组数据的中位数
//   - values: 待统计的数据，默认要求已按升序排列
//   - needSort: 是否需要先升序排序，可选；为 true 时就会地对 values 排序（会修改调用方切片）
//   - 若输入为空切片，返回 0.0
func CalculateMedian[T gokit.Number](values []T, needSort ...bool) float64 {
	if len(values) == 0 {
		return 0
	}

	if len(needSort) > 0 && needSort[0] {
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	}

	n := len(values)
	if n%2 == 1 {
		return float64(values[n/2])
	}

	return float64(values[n/2-1]+values[n/2]) / 2.0
}

// CalculateMean 计算均值
func CalculateMean[T gokit.Number](values ...T) float64 {
	n := len(values)
	if n == 0 {
		return 0.0
	}

	return float64(Sum(values...)) / float64(n)
}

// CalculateAbs 计算绝对值
func CalculateAbs[T gokit.Number](value T) T {
	if value < 0 {
		return -value
	}

	return value
}

// Sum 对一系列数据进行求和
func Sum[T gokit.Number](values ...T) (sum T) {
	for _, val := range values {
		sum += val
	}

	return sum
}

// Count 统计数据中各个值的个数
func Count[T gokit.Number](values ...T) map[T]int {
	count := make(map[T]int)
	if len(values) == 0 {
		return count
	}

	for _, v := range values {
		count[v]++
	}

	return count
}

// CalculateRange 计算数据的取值范围（最小值、最大值）
//   - 若输入为空切片，返回 (0, 0)
func CalculateRange[T gokit.Number](values ...T) (min, max T) {
	if len(values) == 0 {
		return T(0), T(0)
	}

	max, min = values[0], values[0]
	for _, val := range values[1:] {
		if val < min {
			min = val
		}

		if val > max {
			max = val
		}
	}

	return
}
