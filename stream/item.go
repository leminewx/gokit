package stream

type (
	// Item 定义数据流元素的基础接口
	Item[T any] interface {
		// SetValue 设置元素值
		SetValue(value T)
		// SetGroup 设置元素组
		SetGroup(group string)
		// SetIndex 设置元素索引
		SetIndex(index uint64)
		// GetValue 获取元素值
		GetValue() T
		// GetGroup 获取元素组
		GetGroup() string
		// GetIndex 获取元素索引
		GetIndex() uint64
	}

	// item 定义数据流元素的基础结构
	item[T any] struct {
		// index 数据索引
		//	 - 最大索引：18446744073709551615
		//	 - 1秒1亿条数据时，1年的数据量：3162240000000000 = 100000000（秒）* 3600（时）* 24（天）* 366（年）
		//	 - 1秒1亿条数据时，大约使用年限：5833（年）= 18446744073709551615 / 3162240000000000
		index uint64
		group string
		value T
	}
)

// 编译期断言：内置元素实现 Item 接口
var _ Item[int] = (*item[int])(nil)

// NewItem 新建一个基础数据流元素
func NewItem[T any](value T) Item[T] { return &item[T]{value: value} }

// SetValue 设置元素值
func (own *item[T]) SetValue(value T) { own.value = value }

// SetGroup 设置元素组
func (own *item[T]) SetGroup(group string) { own.group = group }

// SetIndex 设置元素索引
func (own *item[T]) SetIndex(index uint64) { own.index = index }

// GetValue 获取元素值
func (own *item[T]) GetValue() T { return own.value }

// GetGroup 获取元素组
func (own *item[T]) GetGroup() string { return own.group }

// GetIndex 获取元素索引
func (own *item[T]) GetIndex() uint64 { return own.index }
