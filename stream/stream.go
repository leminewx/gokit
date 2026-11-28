// Package stream 提供基于 channel 与 goroutine 的泛型数据流处理能力，
// 支持转换、过滤、去重、采样、批处理、多流对齐等链式算子。
//
// 每条数据流都必须及时消费，否则上游会被反压阻塞。
package stream

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"gitee.com/leminewx/gokit/timex"
)

// Stream 数据流接口（每一条数据流要及时消费，否则容易导致阻塞）
//
// 说明：Go 的接口方法不支持泛型参数，因此接口层不提供 Map[U]；需要更换元素类型的
// 场景，请使用 Transform / Concat 或在调用方自行转换。
//
// 取消语义：
//   - 数据源通过 NewStreamOfContext / NewStreamByContext / NewStreamFromLineFile / NewStreamFromJsonFile 绑定 context
//   - 中间算子通过 WithContext 替换取消信号，派生流会自动继承
//   - context 取消后，整条流水线的 goroutine 会逐级退出，Wait / Read 立即返回
type Stream[T any] interface {
	// WithContext 替换当前流派生出的下游算子所使用的取消信号
	//   - 应在构建算子链之前调用，已启动的算子 goroutine 不受影响
	WithContext(context.Context) Stream[T]
	// WithClock 替换当前流派生出的下游算子所使用的时间源，缺省为系统时钟
	WithClock(timex.Clock) Stream[T]
	// Transform 处理
	//   - 零值也会继续向下游流动（与 Map 一致）
	Transform(transform func(item Item[T]) T, callback ...func()) Stream[T]
	// Group 分组
	//   - 为每个元素设置分组，通过 item.GetGroup() 查询，为后续按组消费提供支撑
	Group(func(item Item[T]) string) Stream[T]
	// Concat 聚合
	Concat(...Stream[T]) Stream[T]
	// Filter 过滤
	//   - 满足过滤条件时返回 true（保留）；否则返回 false（丢弃）
	Filter(func(item Item[T]) bool) Stream[T]
	// Deduplicate 去重
	//   - window: 设置一个合理的时间窗口重置去重记录，防止内存无限增长
	Deduplicate(deduplicate func(item Item[T]) any, window ...time.Duration) Stream[T]
	// Peek 窥视
	Peek(peek func(item Item[T]), callback ...func()) Stream[T]
	// TakeFirst 只接收前 num 个元素
	TakeFirst(int) Stream[T]
	// TakeLast 只接收后 num 个元素
	TakeLast(int) Stream[T]
	// SampleByStep 根据步长采样
	SampleByStep(int64) Stream[T]
	// SampleByTime 根据时间采样
	SampleByTime(time.Duration) Stream[T]
	// FanOut 将当前流散出到多个子流（不修改原流）
	//   - num: 子流数量
	//   - 子流不能有任意一阻塞，否则会导致整体阻塞
	FanOut(num int) []Stream[T]
	// WriteToJSON 写入到 JSON 写入器
	//   - writer: JSON 写入器
	//   - 注意: 输出格式为 [obj1, obj2, ...]
	//   - 注意: 这是旁路操作，写入失败也不影响原始数据的流出
	WriteToJSON(writer io.Writer) Stream[T]
	// Consume 消费
	//   - consume: 消费函数
	//   - async: 是否异步执行
	Consume(consume func(item Item[T]), async ...bool)
	// GetChannel 获取数据流管道
	GetChannel() <-chan Item[T]
	// Read 读取单个元素
	Read() (Item[T], bool)
	// Wait 阻塞直到流结束或取消
	Wait()
}

// cacheSize 数据流通道的缺省缓存大小（全局生效）
var cacheSize int32 = 8

// SetStreamCacheSize 设置数据流通道的缓存大小
//   - 建议在构建数据流之前设置，运行期修改只对新创建的通道生效
func SetStreamCacheSize(size int) {
	if size < 0 {
		return
	}

	atomic.StoreInt32(&cacheSize, int32(size))
}

func getCacheSize() int {
	return int(atomic.LoadInt32(&cacheSize))
}
