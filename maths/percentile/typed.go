package percentile

import "gitee.com/leminewx/gokit"

// TypedPercentiler 为 Percentiler 提供泛型输入入口：数据点按调用方指定的数值类型添加，
// 内部统一转换为 float64 参与计算，分位值仍以 float64 返回。
//
// 该类型是既有三种实现的类型安全包装，不改变任何算法口径：
//
//	t := percentile.NewTypedNearestRank[int64]()
//	t.Add(120_000) // 纳秒级耗时，无需先转 float64
//	p99, err := t.CalculatePercentile(99)
//
// 精度边界：计算域为 float64，可精确表示的整数上限为 2^53。超过该范围的 int64 / uint64
// 数据点在 Add 时即被按最近偶数舍入（2^53+1 会变为 2^53），分位结果无法还原原始值；
// 需要完整保留整数精度时，应使用返回样本原值的选择型分位（最近秩），而不是本类型。
//
// 零值不可用，必须通过 NewTypedExact / NewTypedTDigest / NewTypedNearestRank 创建；
// 不同数值类型或不同算法的实例不能合并（前者在编译期即被拒绝，后者返回 ErrNotMergeable）。
type TypedPercentiler[T gokit.Number] struct {
	inner Percentiler
}

// NewTypedExact 创建输入为 T 的精确百分位数计算器（R-7 线性插值）
func NewTypedExact[T gokit.Number]() *TypedPercentiler[T] {
	return &TypedPercentiler[T]{inner: NewExact()}
}

// NewTypedTDigest 创建输入为 T 的 T-Digest 近似百分位数计算器，compression 建议 100~200
func NewTypedTDigest[T gokit.Number](compression float64) *TypedPercentiler[T] {
	return &TypedPercentiler[T]{inner: NewTDigest(compression)}
}

// NewTypedNearestRank 创建输入为 T 的最近秩百分位数计算器（结果为真实观测值，不做插值）
func NewTypedNearestRank[T gokit.Number]() *TypedPercentiler[T] {
	return &TypedPercentiler[T]{inner: NewNearestRank()}
}

// Add 添加一个数据点（内部转换为 float64，超过 2^53 的整数会被舍入）
func (own *TypedPercentiler[T]) Add(value T) {
	own.inner.Add(float64(value))
}

// GetCount 返回当前数据点总数
func (own *TypedPercentiler[T]) GetCount() int {
	return own.inner.GetCount()
}

// Merge 合并另一个同算法、同数值类型的实例
func (own *TypedPercentiler[T]) Merge(other *TypedPercentiler[T]) error {
	if other == nil {
		return ErrNotMergeable
	}
	return own.inner.Merge(other.inner)
}

// CalculatePercentile 计算给定百分位数 p (0 <= p <= 100) 的值
func (own *TypedPercentiler[T]) CalculatePercentile(p float64) (float64, error) {
	return own.inner.CalculatePercentile(p)
}

// GetPercentiler 返回底层的非泛型实例，便于与接受 Percentiler 的既有 API 互操作
func (own *TypedPercentiler[T]) GetPercentiler() Percentiler {
	return own.inner
}
