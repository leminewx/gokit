package cache

import (
	"gitee.com/leminewx/gokit/timex"
)

type (
	// options 缓存构造选项的公共载体，LRU 与 TTL 共享
	options[K comparable, V any] struct {
		source  timex.Clock
		onEvict func(key K, value V)
	}

	// Option 缓存构造选项
	Option[K comparable, V any] func(*options[K, V])
)

// WithClock 指定缓存使用的时间源，默认使用系统时钟
//   - 仅 TTL 缓存使用该时间源；LRU 缓存不依赖时间，传入后会被忽略
//     （保留该选项是为了让 LRU/TTL 共享同一套构造签名）
//   - source 为 nil 时忽略该选项
func WithClock[K comparable, V any](source timex.Clock) Option[K, V] {
	return func(own *options[K, V]) {
		if source != nil {
			own.source = source
		}
	}
}

// WithOnEvict 指定元素被移出缓存时的回调
//   - 触发场景：LRU 容量淘汰，TTL 过期清理与主动删除
//   - 回调在锁外执行，可在回调内安全地操作缓存本身
//   - onEvict 为 nil 时忽略该选项
func WithOnEvict[K comparable, V any](onEvict func(key K, value V)) Option[K, V] {
	return func(own *options[K, V]) {
		if onEvict != nil {
			own.onEvict = onEvict
		}
	}
}

// applyOptions 应用选项并补齐默认值
func applyOptions[K comparable, V any](optionList []Option[K, V]) options[K, V] {
	own := options[K, V]{source: timex.NewSystemClock()}
	for _, option := range optionList {
		option(&own)
	}

	return own
}
