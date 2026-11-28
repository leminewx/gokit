package percentile

import "errors"

const _ERROR_PREFIX = "gokit.maths.percentile: "

var (
	// ErrNotMergeable 表示两个百分位统计器的实现类型不同
	ErrNotMergeable = errors.New(_ERROR_PREFIX + "failed to merge percentilers of different implementations")
	// ErrCompressionMismatch 表示压缩因子不一致，无法合并
	ErrCompressionMismatch = errors.New(_ERROR_PREFIX + "compression factors differ")
	// ErrPercentileOutOfRange 表示百分位不在 [0, 100] 区间
	ErrPercentileOutOfRange = errors.New(_ERROR_PREFIX + "percentile must be between 0 and 100")
	// ErrNoDataPoints 表示没有可用数据点
	ErrNoDataPoints = errors.New(_ERROR_PREFIX + "no data points")
)
