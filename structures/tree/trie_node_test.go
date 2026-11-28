package tree

import "testing"

func TestNewTrieNodeStatic(t *testing.T) {
	node := newTrieNode[int]("users", ":", "*")

	if node.segment != "users" {
		t.Fatalf("segment = %q, want %q", node.segment, "users")
	}
	if node.isWildcard {
		t.Fatal("静态节点不应被识别为通配节点")
	}
	if node.pathname != "" {
		t.Fatalf("静态节点 pathname = %q, want 空", node.pathname)
	}
	if node.priority != 3 {
		t.Fatalf("静态节点优先级 = %d, want 3", node.priority)
	}
	if node.children == nil {
		t.Fatal("子节点切片应已初始化，避免追加时再次分配")
	}
	if len(node.children) != 0 {
		t.Fatalf("新建节点子节点数 = %d, want 0", len(node.children))
	}
}

func TestNewTrieNodeWildcard(t *testing.T) {
	node := newTrieNode[int](":id", ":", "*")

	if !node.isWildcard {
		t.Fatal("参数节点应被识别为通配节点")
	}
	if node.pathname != "id" {
		t.Fatalf("pathname = %q, want %q", node.pathname, "id")
	}
	if node.priority != 2 {
		t.Fatalf("参数节点优先级 = %d, want 2", node.priority)
	}
}

func TestNewTrieNodeMultiply(t *testing.T) {
	node := newTrieNode[int]("*path", ":", "*")

	if !node.isWildcard {
		t.Fatal("多段通配节点应被识别为通配节点")
	}
	if node.pathname != "path" {
		t.Fatalf("pathname = %q, want %q", node.pathname, "path")
	}
	if node.priority != 1 {
		t.Fatalf("多段通配节点优先级 = %d, want 1", node.priority)
	}
}

func TestNewTrieNodeCustomWildcard(t *testing.T) {
	// 通配符字符可被替换，仍能正确提取名称与优先级
	node := newTrieNode[int]("{id}", "{", "}")

	if !node.isWildcard {
		t.Fatal("自定义通配符节点应被识别为通配节点")
	}
	if node.priority != 2 {
		t.Fatalf("优先级 = %d, want 2", node.priority)
	}
}

func TestTrieNodePriorityOrdering(t *testing.T) {
	// 优先级：静态(3) > 单层通配(2) > 多段通配(1)
	static := newTrieNode[int]("users", ":", "*")
	wildcard := newTrieNode[int](":id", ":", "*")
	multiply := newTrieNode[int]("*path", ":", "*")

	if !(static.priority > wildcard.priority && wildcard.priority > multiply.priority) {
		t.Fatalf("优先级顺序错误: static=%d wildcard=%d multiply=%d",
			static.priority, wildcard.priority, multiply.priority)
	}
}
