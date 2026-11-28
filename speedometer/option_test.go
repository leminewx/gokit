package speedometer

import (
	"testing"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

func TestWithClock(t *testing.T) {
	source := timex.NewManual(time.Unix(1700000000, 0))

	own := &defaultSpeedometer{}
	WithClock(source)(own)
	if own.clock != source {
		t.Fatal("WithClock 未替换时间源")
	}

	// nil 时间源应被忽略，保留原值
	WithClock(nil)(own)
	if own.clock != source {
		t.Fatal("nil 时间源不应覆盖原值")
	}
}

func TestWithBucketNumber(t *testing.T) {
	own := &defaultSpeedometer{bucketNumber: _DEFAULT_BUCKET_NUMBER}
	WithBucketNumber(10)(own)
	if own.bucketNumber != 10 {
		t.Fatalf("bucketNumber = %d, want 10", own.bucketNumber)
	}

	// 非正桶数量应被忽略
	WithBucketNumber(0)(own)
	if own.bucketNumber != 10 {
		t.Fatalf("非正桶数量不应覆盖原值: %d", own.bucketNumber)
	}
}

func TestNewDefaultBucketNumber(t *testing.T) {
	if got := _DEFAULT_BUCKET_NUMBER; got != 7 {
		t.Fatalf("默认桶数量 = %d, want 7", got)
	}

	// New 应用选项后应保留指定桶数量
	speedometer := New(time.Second, WithBucketNumber(10))
	if speedometer == nil {
		t.Fatal("New 返回 nil")
	}
}

func TestNewUsesDefaultClock(t *testing.T) {
	own, ok := New(time.Second).(*defaultSpeedometer)
	if !ok {
		t.Fatal("New 应返回 *defaultSpeedometer")
	}
	if own.clock == nil {
		t.Fatal("默认应使用系统时钟")
	}
}
