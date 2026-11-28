package stringx

import (
	"errors"
	"testing"
)

func TestJoinInts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		items []int
		sep   string
		want  string
	}{
		{name: "空切片", items: nil, sep: ",", want: ""},
		{name: "单元素", items: []int{7}, sep: ",", want: "7"},
		{name: "多元素", items: []int{1, 2, 3}, sep: ",", want: "1,2,3"},
		{name: "负数与自定义分隔符", items: []int{-1, 0, 1}, sep: " | ", want: "-1 | 0 | 1"},
	}

	for _, tc := range cases {
		if got := JoinInts(tc.items, tc.sep); got != tc.want {
			t.Errorf("%s: JoinInts(%v, %q) = %q, want %q", tc.name, tc.items, tc.sep, got, tc.want)
		}
	}

	if got := JoinInts([]int8{-128, 127}, ","); got != "-128,127" {
		t.Errorf("JoinInts(int8) = %q, want %q", got, "-128,127")
	}
}

func TestJoinUints(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		items []uint
		sep   string
		want  string
	}{
		{name: "空切片", items: nil, sep: ",", want: ""},
		{name: "单元素", items: []uint{9}, sep: ",", want: "9"},
		{name: "多元素", items: []uint{4, 5, 6}, sep: "-", want: "4-5-6"},
	}

	for _, tc := range cases {
		if got := JoinUints(tc.items, tc.sep); got != tc.want {
			t.Errorf("%s: JoinUints(%v, %q) = %q, want %q", tc.name, tc.items, tc.sep, got, tc.want)
		}
	}

	if got := JoinUints([]uint8{0, 255}, ","); got != "0,255" {
		t.Errorf("JoinUints(uint8) = %q, want %q", got, "0,255")
	}
}

func TestParseNumbers(t *testing.T) {
	t.Parallel()

	got, err := ParseNumbers[int]([]string{"1", " -2 ", "3"})
	if err != nil {
		t.Fatalf("ParseNumbers[int] 返回错误: %v", err)
	}
	want := []int{1, -2, 3}
	if len(got) != len(want) {
		t.Fatalf("ParseNumbers[int] 长度 = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ParseNumbers[int][%d] = %d, want %d", i, got[i], want[i])
		}
	}

	empty, err := ParseNumbers[int64](nil)
	if err != nil {
		t.Fatalf("ParseNumbers[int64](nil) 返回错误: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("ParseNumbers[int64](nil) 长度 = %d, want 0", len(empty))
	}
}

func TestParseNumbersBoundary(t *testing.T) {
	t.Parallel()

	if got, err := ParseNumbers[int8]([]string{"127", "-128"}); err != nil {
		t.Errorf("ParseNumbers[int8] 边界值返回错误: %v", err)
	} else if len(got) != 2 || got[0] != 127 || got[1] != -128 {
		t.Errorf("ParseNumbers[int8] 边界值 = %v, want [127 -128]", got)
	}

	if _, err := ParseNumbers[int8]([]string{"128"}); !errors.Is(err, ErrInvalidNumber) {
		t.Errorf("ParseNumbers[int8](\"128\") 错误 = %v, want ErrInvalidNumber", err)
	}

	if got, err := ParseNumbers[uint8]([]string{"0", "255"}); err != nil {
		t.Errorf("ParseNumbers[uint8] 边界值返回错误: %v", err)
	} else if len(got) != 2 || got[0] != 0 || got[1] != 255 {
		t.Errorf("ParseNumbers[uint8] 边界值 = %v, want [0 255]", got)
	}

	if _, err := ParseNumbers[uint8]([]string{"256"}); !errors.Is(err, ErrInvalidNumber) {
		t.Errorf("ParseNumbers[uint8](\"256\") 错误 = %v, want ErrInvalidNumber", err)
	}

	if _, err := ParseNumbers[uint8]([]string{"-1"}); !errors.Is(err, ErrInvalidNumber) {
		t.Errorf("ParseNumbers[uint8](\"-1\") 错误 = %v, want ErrInvalidNumber", err)
	}

	if _, err := ParseNumbers[int]([]string{"1", "abc"}); !errors.Is(err, ErrInvalidNumber) {
		t.Errorf("ParseNumbers[int](\"abc\") 错误 = %v, want ErrInvalidNumber", err)
	}
}

func BenchmarkParseNumbers(b *testing.B) {
	items := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := ParseNumbers[int](items); err != nil {
			b.Fatalf("ParseNumbers 返回错误: %v", err)
		}
	}
}
