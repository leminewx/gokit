package tree

import "strings"

// trieNode 字典树节点
type trieNode[T any] struct {
	// 优先级（1-通配符；2-参数；3-静态）
	priority int
	// 节点路径
	segment string
	// 是否是页节点（叶节点也可以存在子节点）
	isLeaf bool
	// 是否是通配节点
	isWildcard bool
	// 通配名称
	pathname string
	// 节点数据
	value T
	// 子节点
	children []*trieNode[T]
}

// newTrieNode 新建一个字典树节点
func newTrieNode[T any](segment string, wildcard, multiply string) *trieNode[T] {
	// 创建节点
	node := &trieNode[T]{
		segment:    segment,
		isWildcard: strings.HasPrefix(segment, wildcard) || strings.HasPrefix(segment, multiply),
		children:   make([]*trieNode[T], 0),
	}

	// 设置优先级：3（静态节点）> 2（单层通配节点）> 1（多层通配节点）
	switch {
	case strings.HasPrefix(node.segment, multiply):
		node.priority = 1
	case strings.HasPrefix(node.segment, wildcard):
		node.priority = 2
	default:
		node.priority = 3
	}

	// 提取通配名称
	if node.isWildcard {
		if strings.HasPrefix(segment, multiply) {
			node.pathname = strings.Split(segment, multiply)[1]
		}

		if strings.HasPrefix(segment, wildcard) {
			node.pathname = strings.Split(segment, wildcard)[1]
		}
	}

	return node
}
