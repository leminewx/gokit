package stringx

import (
	"strings"
	"sync"
)

// _MAX_POOLED_BUILDER_CAP 限制回收到池中的缓冲容量，避免长期驻留大对象
const _MAX_POOLED_BUILDER_CAP = 64 << 10 // 64 KiB

var stringBuilderPool = sync.Pool{
	New: func() any { return new(strings.Builder) },
}

// GetBuilder 获取一个新的 strings.Builder 实例
func GetBuilder() *strings.Builder {
	return stringBuilderPool.Get().(*strings.Builder)
}

// PutBuilder 将 strings.Builder 实例放回池中
//   - 传入 nil 时直接忽略
//   - 容量超过 64 KiB 的实例不再复用，交由 GC 回收
func PutBuilder(builder *strings.Builder) {
	if builder == nil || builder.Cap() > _MAX_POOLED_BUILDER_CAP {
		return
	}

	builder.Reset()
	stringBuilderPool.Put(builder)
}
