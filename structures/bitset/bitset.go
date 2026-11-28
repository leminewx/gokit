package bitset

import (
	"bytes"
	"fmt"
	"math/bits"
)

const (
	_MASK  = 0x3F // 63的二进制表示
	_SHIFT = 6    // 2^6 = 64
)

// BitSet 定义位集合的数据结构
type BitSet struct {
	data []uint64
	size int
}

// New 创建一个新的BitSet，初始容量为n位
func New(n int) *BitSet {
	if n < 0 {
		n = 0
	}
	return &BitSet{
		data: make([]uint64, (n+63)/64),
		size: n,
	}
}

// Set 将指定位置的位设置为1
func (own *BitSet) Set(pos int) *BitSet {
	if pos < 0 || pos >= own.size {
		return own
	}
	idx := pos >> _SHIFT
	own.data[idx] |= 1 << (pos & _MASK)
	return own
}

// Clear 将指定位置的位设置为0
func (own *BitSet) Clear(pos int) *BitSet {
	if pos < 0 || pos >= own.size {
		return own
	}
	idx := pos >> _SHIFT
	own.data[idx] &^= 1 << (pos & _MASK)
	return own
}

// Has 检查指定位置的位是否为1
func (own *BitSet) Has(pos int) bool {
	if pos < 0 || pos >= own.size {
		return false
	}
	idx := pos >> _SHIFT
	return own.data[idx]&(1<<(pos&_MASK)) != 0
}

// Resize 调整BitSet的大小
func (own *BitSet) Resize(newSize int) *BitSet {
	if newSize < 0 {
		newSize = 0
	}

	newLen := (newSize + 63) / 64
	switch {
	case newLen > len(own.data):
		newData := make([]uint64, newLen)
		copy(newData, own.data)
		own.data = newData
	case newLen < len(own.data):
		own.data = own.data[:newLen]
	}

	// 缩小位数时需清除最后一个 word 中超出的高位。
	// 注意不能只放在 newLen < 旧长度 的分支里：64 位缩到 1 位时 newLen 仍为 1，
	// 长度不变但第 1 位以后的数据必须被清除，否则会把已越界的位置算进 GetCount/Has。
	if newLen > 0 && newSize < own.size {
		validBits := newSize % 64
		if validBits == 0 {
			validBits = 64
		}
		lowMask := ^uint64(0) >> (64 - validBits)
		own.data[newLen-1] &= lowMask
	}

	own.size = newSize
	return own
}

// GetLength 返回 BitSet 的位数
func (own *BitSet) GetLength() int {
	return own.size
}

// GetCount 返回 BitSet 中 1 的数量
func (own *BitSet) GetCount() int {
	count := 0
	for _, v := range own.data {
		count += bits.OnesCount64(v)
	}
	return count
}

// Union 计算当前BitSet与另一个BitSet的并集
//   - other 比当前更大时先扩容，避免超出当前位数的置位被静默丢弃
func (own *BitSet) Union(other *BitSet) *BitSet {
	if other == nil {
		return own
	}

	if other.size > own.size {
		own.Resize(other.size)
	}

	for i := 0; i < len(other.data) && i < len(own.data); i++ {
		own.data[i] |= other.data[i]
	}

	return own
}

// Intersect 计算当前BitSet与另一个BitSet的交集
func (own *BitSet) Intersect(other *BitSet) *BitSet {
	minLen := min(len(own.data), len(other.data))
	for i := 0; i < minLen; i++ {
		own.data[i] &= other.data[i]
	}
	// 清除超出other长度的位
	for i := minLen; i < len(own.data); i++ {
		own.data[i] = 0
	}
	return own
}

// Subtract 计算当前BitSet与另一个BitSet的差集
func (own *BitSet) Subtract(other *BitSet) *BitSet {
	minLen := min(len(own.data), len(other.data))
	for i := 0; i < minLen; i++ {
		own.data[i] &^= other.data[i]
	}
	return own
}

// Complement 计算BitSet的补集
func (own *BitSet) Complement() *BitSet {
	for i := range own.data {
		own.data[i] = ^own.data[i]
	}
	// 清除超出size的高位
	if own.size > 0 {
		lastIdx := (own.size - 1) >> _SHIFT
		validBits := own.size % 64
		if validBits == 0 {
			validBits = 64
		}
		lowMask := ^uint64(0) >> (64 - validBits)
		own.data[lastIdx] &= lowMask
	}
	return own
}

// Clone 创建BitSet的副本
func (own *BitSet) Clone() *BitSet {
	dataCopy := make([]uint64, len(own.data))
	copy(dataCopy, own.data)
	return &BitSet{
		data: dataCopy,
		size: own.size,
	}
}

// IsEqualTo 检查两个BitSet是否相等
func (own *BitSet) IsEqualTo(other *BitSet) bool {
	if own.size != other.size {
		return false
	}

	minLen := min(len(own.data), len(other.data))
	for i := 0; i < minLen; i++ {
		if own.data[i] != other.data[i] {
			return false
		}
	}

	// 检查较长BitSet的剩余部分是否全为0
	if len(own.data) > minLen {
		for i := minLen; i < len(own.data); i++ {
			if own.data[i] != 0 {
				return false
			}
		}
	} else if len(other.data) > minLen {
		for i := minLen; i < len(other.data); i++ {
			if other.data[i] != 0 {
				return false
			}
		}
	}

	return true
}

// String 返回BitSet的字符串表示形式
func (own *BitSet) String() string {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i := 0; i < own.size; i++ {
		if own.Has(i) {
			fmt.Fprintf(&buf, "%d ", i)
		}
	}
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1) // 移除最后一个空格
	}
	buf.WriteByte(']')
	return buf.String()
}

// ForEach 遍历所有设置为1的位，并对每个位执行回调函数
func (own *BitSet) ForEach(fn func(pos int) bool) {
	for i, v := range own.data {
		if v == 0 {
			continue
		}
		base := i << _SHIFT
		for j := 0; j < 64; j++ {
			if v&(1<<j) != 0 {
				pos := base + j
				if pos >= own.size {
					break
				}
				if !fn(pos) {
					return
				}
			}
		}
	}
}
