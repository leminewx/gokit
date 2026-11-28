package tree

import (
	"sort"
	"strings"
)

// Trie 字典树
type Trie[T any] struct {
	// 字典路径的分隔符，默认使用 /
	delim string
	// 单层通配符，默认使用 :
	wildcard string
	// 多层通配符，默认使用 *
	multiply string
	//	根节点
	root *trieNode[T]
}

// NewTrie 新建一个字典树
//   - 默认分割符：/
//   - 默认单层通配符：:
//   - 默认多层通配符：*
func NewTrie[T any]() *Trie[T] {
	return &Trie[T]{
		delim:    "/",
		wildcard: ":",
		multiply: "*",
		root:     &trieNode[T]{},
	}
}

// SetDelim 设置路径分隔符
func (own *Trie[T]) SetDelim(delim string) { own.delim = delim }

// SetWildcard 设置单层通配符
func (own *Trie[T]) SetWildcard(char string) { own.wildcard = char }

// SetMultiWildcard 设置多层通配符
func (own *Trie[T]) SetMultiWildcard(char string) { own.multiply = char }

// Insert 插入节点
func (own *Trie[T]) Insert(pattern string, value T) {
	segments := own.splitPattern(pattern)
	own.insert(own.root, segments, value, 0)
}

// insert 递归插入路由到前缀树
func (own *Trie[T]) insert(node *trieNode[T], segments []string, value T, depth int) {
	// 到达路径末端（叶子）
	if depth == len(segments) {
		node.isLeaf = true
		node.value = value
		return
	}

	// 获取当前路径片段
	segment := segments[depth]

	// 查找匹配的子节点
	var child *trieNode[T]
	for _, c := range node.children {
		if c.segment == segment {
			child = c
			break
		}
	}

	// 如果没有找到匹配的子节点，创建新节点
	if child == nil {
		child = newTrieNode[T](segment, own.wildcard, own.multiply) // 创建新节点
		node.children = append(node.children, child)

		// 按优先级排序：静态路径 > 单层通配 > 多层通配
		sort.Slice(node.children, func(i, j int) bool {
			return node.children[i].priority > node.children[j].priority
		})
	}

	// 递归插入
	own.insert(child, segments, value, depth+1)
}

// Get 获取与指定模式完全匹配的节点值（不进行通配符匹配）
//   - 用于在同一个模式上叠加注册内容，例如向同一主题追加订阅回调
func (own *Trie[T]) Get(pattern string) (T, bool) {
	segments := own.splitPattern(pattern)
	node := own.root
	for _, segment := range segments {
		var next *trieNode[T]
		for _, child := range node.children {
			if child.segment == segment {
				next = child
				break
			}
		}

		if next == nil {
			return *new(T), false
		}

		node = next
	}

	if !node.isLeaf {
		return *new(T), false
	}

	return node.value, true
}

// Search 搜索匹配的路由
func (own *Trie[T]) Search(path string) (T, map[string]string, bool) {
	segments := own.splitPattern(path)
	params := make(map[string]string)
	if node, isLeaf := own.search(own.root, segments, params, 0); isLeaf {
		return node.value, params, true
	}

	return *new(T), nil, false
}

// search 递归搜索匹配的路由
func (own *Trie[T]) search(node *trieNode[T], segments []string, params map[string]string, depth int) (*trieNode[T], bool) {
	// 当前深度与路径深度匹配，则返回当前节点
	if depth == len(segments) {
		return node, node.isLeaf
	}

	segment := segments[depth]
	children := node.children
	for i := 0; i < len(children); i++ {
		child := children[i]

		// 优先尝试精确匹配
		if !child.isWildcard {
			// 匹配失败，直接跳过
			if child.segment != segment {
				continue
			}

			// 匹配成功，进入下一轮递归搜索
			if n, ok := own.search(child, segments, params, depth+1); ok {
				return n, n.isLeaf
			}
		}

		// 尝试参数匹配 :param
		if child.priority == 2 {
			if child.pathname != "" {
				params[child.pathname] = segment
			}

			if n, ok := own.search(child, segments, params, depth+1); ok {
				return n, n.isLeaf
			}

			// 回溯时删除参数
			delete(params, child.pathname)
			continue
		}

		// 尝试通配符匹配 *param 或 *
		if child.priority == 1 {
			// 通配符匹配，收集剩余路径（使用自身配置的分隔符，而非硬编码的 /）
			if depth < len(segments) {
				remaining := strings.Join(segments[depth:], own.delim)
				if child.pathname != "" {
					params[child.pathname] = remaining
				}
			}

			return child, child.isLeaf
		}
	}

	return nil, false
}

// splitPattern SplitPattern 分割模式
func (own *Trie[T]) splitPattern(pattern string) []string {
	segments := strings.Split(strings.Trim(pattern, own.delim), own.delim)
	result := make([]string, 0, len(segments))
	for _, seg := range segments {
		if seg != "" {
			result = append(result, seg)
		}
	}

	return result
}
