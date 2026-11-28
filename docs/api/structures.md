# structures 数据结构 API

导入前缀：`gitee.com/leminewx/gokit/structures/...`

## structures/dict

四种实现共享 `Dict[K comparable, V any]` 接口。

| 构造函数 | 类型 | 线程安全 | 适用场景 |
|----------|------|----------|----------|
| `NewUnsafeDict[K,V]() *UnsafeDict[K,V]` | `map[K]V` | 否 | 单协程、追求最高性能 |
| `NewSyncDict[K,V]() *SyncDict[K,V]` | 封装 map + 锁 | 是 | 通用并发场景 |
| `NewSafeDict[K,V]() *SafeDict[K,V]` | 安全字典 | 是 | 读多写少 |
| `NewShardSafeDict[K,V]() *ShardSafeDict[K,V]` | 分片字典 | 是 | 高并发、降低锁竞争 |

接口方法：

| 方法 | 说明 |
|------|------|
| `Set(key K, val V)` / `Get(key K) (V, bool)` | 写入 / 读取 |
| `Update(map[K]V)` | 批量写入 |
| `Delete(...K)` | 批量删除 |
| `HasAll(...K) bool` / `HasAny(...K) bool` | 全部存在 / 任一存在 |
| `GetKeys() []K` / `GetValues() []V` / `GetMap() map[K]V` | 快照 |
| `GetItems() iter.Seq2[K, V]` | 迭代器视图，可直接用于 `for-range` |
| `GetLength() int` | 元素个数 |
| `Transform(func(K, V) V)` | 就地更新所有值 |
| `Filter(func(K, V) bool) map[K]V` | 删除不满足条件的项并返回被删除项 |
| `Range(func(K, V))` | 遍历 |
| `Clear()` | 清空 |

迭代器辅助函数（`structures/dict`）：

| 函数 | 说明 |
|------|------|
| `ExtractKeys[K, V](iter.Seq2[K, V]) iter.Seq[K]` | 从键值对迭代器提取键，常与 `GetItems` 配合 |
| `ExtractValues[K, V](iter.Seq2[K, V]) iter.Seq[V]` | 从键值对迭代器提取值 |
| `Collect[K, V](iter.Seq2[K, V]) *UnsafeDict[K, V]` | 将键值对迭代器收集为非并发安全字典 |

```go
for key := range dict.ExtractKeys(d.GetItems()) {
	fmt.Println(key)
}
```

## structures/set

| 构造函数 | 类型 | 线程安全 |
|----------|------|----------|
| `NewUnsafeSet[T](elems ...T) Set[T]` | `map[T]struct{}` | 否 |
| `NewSyncSet[T](elems ...T) Set[T]` | `sync.Map` | 是（读多写少） |
| `NewSafeSet[T](elems ...T) Set[T]` | 加锁集合 | 是 |

`ConvertToUnsafeSet` / `ConvertToSyncSet` / `ConvertToSafeSet` 可在实现之间转换。

| 方法 | 说明 |
|------|------|
| `Add(...T)` / `Delete(...T)` | 增删 |
| `HasAll(...T) bool` / `HasAny(...T) bool` | 包含判断 |
| `GetSlice() []T` / `GetMap() map[T]struct{}` / `GetLength() int` | 快照 |
| `GetAll() iter.Seq[T]` | 迭代器视图，可直接用于 `for-range` |
| `IsEmpty()` / `IsEqualTo(Set[T])` / `IsSubsetOf(Set[T])` / `IsSupersetOf(Set[T])` | 关系判断 |
| `Unite` / `Intersect` / `Complement` | 并集 / 交集 / 差集，返回新集合 |
| `Extract(fn) []T` | 提取满足条件的元素，不删除 |
| `Filter(fn) []T` | 提取并删除满足条件的元素 |
| `Range(fn)` / `RangeUntil(fn) bool` | 遍历 / 可中断遍历 |
| `Clone() Set[T]` / `Clear()` | 克隆 / 清空 |

迭代器辅助函数（`structures/set`）：

| 函数 | 说明 |
|------|------|
| `CollectSet[T](iter.Seq[T]) Set[T]` | 由迭代器构建非并发安全集合，常与 `GetAll` 配合 |
| `Union[T](sets ...Set[T]) Set[T]` | 并集，返回新集合 |
| `Intersect[T](first Set[T], rest ...Set[T]) Set[T]` | 交集，返回新集合 |

```go
for elem := range s.GetAll() {
	fmt.Println(elem)
}
merged := set.Union(a, b)
```

## structures/bitset

```go
b := bitset.New(128)
b.Set(3).Set(64)
fmt.Println(b.Has(3), b.GetCount(), b.GetLength())
c := b.Clone().Complement()
u := b.Union(c)
```

| 方法 | 说明 |
|------|------|
| `New(n int) *BitSet` | 指定位数 |
| `Set(pos int)` / `Clear(pos int)` | 置位 / 清位，返回自身以支持链式 |
| `Has(pos int) bool` | 查询位 |
| `GetCount() int` / `GetLength() int` | 置位数量 / 容量 |
| `Union` / `Intersect` / `Subtract` / `Complement` | 集合运算，返回新位集 |
| `IsEqualTo(other *BitSet) bool` | 相等比较 |
| `Clone()` / `Resize(newSize int)` | 复制 / 调整容量 |
| `ForEach(fn func(pos int) bool)` | 遍历置位，`fn` 返回 `false` 停止 |
| `String() string` | 文本表示 |

`BitSet` 不保证并发安全。

## structures/bloom

```go
bf := bloom.NewBloomFilter[string](bloom.Config{
    ExpectedSize:      100_000,
    FalsePositiveRate: 0.01,
    Concurrent:        true,
}, func(s string) []byte { return []byte(s) })

bf.Add("user:1")
if bf.Has("user:1") { /* 可能存在 */ }
fmt.Println(bf.EstimateCount(), bf.CalculateFalsePositiveRate())
```

| 符号 | 说明 |
|------|------|
| `NewBloomFilter[T](cfg Config, toBytes func(T) []byte) BloomFilter[T]` | 创建 |
| `Config.ExpectedSize uint64` | 预期元素数量 |
| `Config.FalsePositiveRate float64` | 期望假阳性率 |
| `Config.HashFunctions []HashFunc` | 自定义哈希（空则用 FNV） |
| `Config.Concurrent bool` | 是否加读锁（位数组本身为原子操作） |
| 接口方法 | `Add(T)`、`Has(T) bool`、`Clear()`、`EstimateCount() uint64`、`CalculateFalsePositiveRate() float64` |
| `HashFunc func() hash.Hash64` | 哈希工厂 |

只允许假阳性，不会出现假阴性。

## structures/tree

Trie 前缀树，支持路径参数匹配。

```go
t := tree.NewTrie[string]()
t.Insert("/api/users/:id", "get-user") // : 单层通配
t.Insert("/static/*", "static")        // * 多层通配

v, params, ok := t.Search("/api/users/42")
// v == "get-user", params == {"id": "42"}, ok == true
```

| 符号 | 说明 |
|------|------|
| `NewTrie[T]() *Trie[T]` | 创建 |
| `Insert(path string, value T)` | 注册模式 |
| `Search(path string) (value T, params map[string]string, exists bool)` | 匹配并返回路径参数 |
| `Tree[T]` 接口 | 上述两个方法的接口形式 |

分隔符 `/`，单层通配 `:`，多层通配 `*`。Trie 非并发安全。

## structures/fsm

泛型有限状态机。

```go
machine := fsm.New[EventData](fsm.WithStopOnError[EventData](true))
machine.AddStates("idle", "running", "done")
_ = machine.AddTransition("idle", "start", "running")
_ = machine.CallOnEnter("running", func(state fsm.State, data EventData) error {
    return nil
})

next, err := machine.Trigger("idle", "start", data)
```

| 符号 | 说明 |
|------|------|
| `New[T](opts ...Option[T]) *FSM[T]` | 创建，默认所有回调错误即中止转移 |
| `AddStates(...State)` / `AddTransition(from State, event Event, to State) error` | 注册状态与转移 |
| `MustAddTransition(...) error` | 注册并返回错误（便于测试断言） |
| `RemoveTransition(from State, event Event) error` | 删除转移 |
| `Trigger(current State, event Event, data T) (State, error)` | 触发转移 |
| `CallOnEnter` / `CallOnEvent` / `CallOnLeave` | 注册回调 |
| `Validate() error` / `HasState(State) bool` | 校验 / 查询 |
| `GetStates` / `GetTransitions` / `GetStats` | 快照与统计 |
| `Export()` / `Import(states, transitions)` | 导出 / 导入（可持久化） |
| `Option` | `WithStopOnError`、`WithEnterErrorPolicy`、`WithEventErrorPolicy`、`WithLeaveErrorPolicy` |
| 回调类型 | `OnEnterHandler[T]`、`OnEventHandler[T]`、`OnLeaveHandler[T]` |

`FSM` 非并发安全，同一实例需串行 `Trigger`。

## 已迁移

时间轮与延迟队列属于主动调度，已移至 [scheduler.md](./scheduler.md)。
