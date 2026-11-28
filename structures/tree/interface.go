package tree

// Tree 树接口
type Tree[T any] interface {
	// Insert 插入
	Insert(path string, value T)
	// Search 搜索
	Search(path string) (value T, params map[string]string, exists bool)
}

// 编译期断言：Trie 实现 Tree 接口
var _ Tree[int] = (*Trie[int])(nil)
