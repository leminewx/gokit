package speedometer

import (
	"gitee.com/leminewx/gokit/timex"
)

// _DEFAULT_BUCKET_NUMBER 缺省的桶数量
const _DEFAULT_BUCKET_NUMBER = 7

// Option 定义计速器的构造选项
type Option func(*defaultSpeedometer)

// WithClock 指定计速器使用的时间源，缺省使用系统时钟
func WithClock(source timex.Clock) Option {
	return func(own *defaultSpeedometer) {
		if source != nil {
			own.clock = source
		}
	}
}

// WithBucketNumber 指定滑动窗口划分的桶数量，缺省 7，建议取值 5~10
//   - 桶数量越多，窗口滑动越平滑，但内存占用与淘汰开销也随之增加
func WithBucketNumber(number int) Option {
	return func(own *defaultSpeedometer) {
		if number > 0 {
			own.bucketNumber = number
		}
	}
}
