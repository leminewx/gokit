package bitset

import (
	"fmt"
	"testing"
)

func TestBitset(t *testing.T) {
	// 创建一个包含100位的BitSet
	bs := New(100)

	// 设置一些位
	bs.Set(10).Set(20).Set(30)

	// 测试位
	fmt.Println("位20是否设置:", bs.Has(20)) // 输出: true
	fmt.Println("位21是否设置:", bs.Has(21)) // 输出: false

	// 统计1的数量
	fmt.Println("设置的位数量:", bs.GetCount()) // 输出: 3

	// 创建另一个BitSet并执行交集操作
	bs2 := New(100)
	bs2.Set(20).Set(40)

	bs.Intersect(bs2)
	fmt.Println("交集后的位:", bs) // 输出: [20]

	// 遍历所有设置的位
	bs.ForEach(func(pos int) bool {
		fmt.Println("找到设置的位:", pos)
		return true // 返回true继续遍历，返回false停止遍历
	})
}

// TestBitSetUnionGrowsToOther 验证 other 更长时并集不会丢弃高位
//   - 回归点：Union 曾只按较短长度合并，超出部分被静默丢弃且 size 不变
func TestBitSetUnionGrowsToOther(t *testing.T) {
	small := New(64)
	small.Set(1)

	large := New(256)
	large.Set(1)
	large.Set(200)

	small.Union(large)

	if small.GetLength() != 256 {
		t.Fatalf("Union 后长度 = %d, want 256", small.GetLength())
	}
	for _, pos := range []int{1, 200} {
		if !small.Has(pos) {
			t.Fatalf("Union 后位 %d 应为 1", pos)
		}
	}
	if got := small.GetCount(); got != 2 {
		t.Fatalf("Union 后位数 = %d, want 2", got)
	}

	// 反向合并仍不应改变长度
	large2 := New(64)
	large2.Set(3)
	large.Union(large2)
	if large.GetLength() != 256 || !large.Has(3) {
		t.Fatalf("小集合合入大集合后结果不正确: len=%d, bit3=%v", large.GetLength(), large.Has(3))
	}
}

// TestBitSetCountComplement 验证跨 uint64 边界的计数与补集
//   - 回归点：Count 使用手写 SWAR popcount，补集/缩容使用高位屏蔽，均需保证边界正确
func TestBitSetCountComplement(t *testing.T) {
	bs := New(130)
	for _, pos := range []int{0, 63, 64, 129} {
		bs.Set(pos)
	}
	if got := bs.GetCount(); got != 4 {
		t.Fatalf("Count = %d, want 4", got)
	}

	bs.Complement()
	if got := bs.GetCount(); got != 130-4 {
		t.Fatalf("Complement 后 Count = %d, want %d", got, 130-4)
	}
	// 补集后不应越界置位
	if bs.Has(130) || bs.Has(-1) {
		t.Fatal("Complement 后越界位应为 0")
	}

	// 缩容时需清除超出新大小的高位
	shrink := New(64)
	shrink.Set(0).Set(63)
	shrink.Resize(1)
	if got := shrink.GetCount(); got != 1 || !shrink.Has(0) {
		t.Fatalf("Resize(1) 后 Count = %d, bit0 = %v; want 1, true", got, shrink.Has(0))
	}

	// 边界：64 位时低位掩码必须是全 1（不能因1<<64 溢出而清空）
	full := New(64)
	full.Set(63)
	full.Resize(64)
	if !full.Has(63) {
		t.Fatal("Resize(64) 不应清除第 63 位")
	}
}

// TestBitSetOperations 覆盖交集/差集/克隆/相等性/遍历/字符串表示与扩容
func TestBitSetOperations(t *testing.T) {
	left := New(100)
	left.Set(1).Set(2).Set(70)

	right := New(100)
	right.Set(2).Set(3).Set(70)

	// Intersect：只保留公共位
	if got := left.Clone().Intersect(right).String(); got != "[2 70]" {
		t.Fatalf("Intersect = %q, want [2 70]", got)
	}

	// Intersect 需清除超出 other 长度的高位
	short := New(64)
	short.Set(3)
	long := New(128)
	long.Set(3).Set(100)
	if got := long.Clone().Intersect(short).GetCount(); got != 1 {
		t.Fatalf("Intersect with shorter other = %d, want 1", got)
	}

	// Subtract：剔除公共位
	if got := left.Clone().Subtract(right).String(); got != "[1]" {
		t.Fatalf("Subtract = %q, want [1]", got)
	}

	// Clone 与 IsEqualTo
	clone := left.Clone()
	if !left.IsEqualTo(clone) {
		t.Fatal("clone should equal source")
	}
	clone.Set(5)
	if left.IsEqualTo(clone) {
		t.Fatal("modified clone should differ")
	}
	// 长度不同直接不相等
	if left.IsEqualTo(New(200)) {
		t.Fatal("different size should not be equal")
	}

	// ForEach：早退后不再回调
	var visited []int
	left.ForEach(func(pos int) bool {
		visited = append(visited, pos)
		return len(visited) < 2
	})
	if len(visited) != 2 || visited[0] != 1 || visited[1] != 2 {
		t.Fatalf("ForEach visited = %v, want [1 2]", visited)
	}

	// String 对空集与跨字边界
	if got := New(10).String(); got != "[]" {
		t.Fatalf("empty String = %q, want []", got)
	}
	boundary := New(130)
	boundary.Set(64).Set(129)
	if got := boundary.String(); got != "[64 129]" {
		t.Fatalf("boundary String = %q, want [64 129]", got)
	}

	// 扩容保留已有位，缩容清高位
	grow := New(8)
	grow.Set(7)
	grow.Resize(128)
	if grow.GetLength() != 128 || !grow.Has(7) || grow.GetCount() != 1 {
		t.Fatalf("Resize grow: len=%d, count=%d", grow.GetLength(), grow.GetCount())
	}

	// 越界读写是安全的 no-op
	if New(4).Set(-1).Set(4).Has(4) {
		t.Fatal("out of range ops should be no-op")
	}
	// New(负数) 视同 0
	if got := New(-5).GetLength(); got != 0 {
		t.Fatalf("New(-5).GetLength() = %d, want 0", got)
	}
}
