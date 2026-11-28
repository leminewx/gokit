package color

import "testing"

func TestCodeString(t *testing.T) {
	tests := []struct {
		name string
		in   Code
		want string
	}{
		{"加粗", STYLE_BOLD, "1"},
		{"淡色", STYLE_DIM, "2"},
		{"删除线", STYLE_STRIKETHROUGH, "9"},
		{"红前景", FG_RED, "31"},
		{"白前景", FG_WHITE, "37"},
		{"亮白前景", FG_BRIGHT_WHITE, "97"},
		{"蓝背景", BG_BLUE, "44"},
		{"亮蓝背景", BG_BRIGHT_BLUE, "104"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Fatalf("Code(%d).String() = %q, want %q", int(tt.in), got, tt.want)
			}
		})
	}
}

func TestCodeConstantRanges(t *testing.T) {
	if STYLE_BOLD != 1 {
		t.Fatalf("STYLE_BOLD = %d, want 1", STYLE_BOLD)
	}
	if FG_BLACK != 30 {
		t.Fatalf("FG_BLACK = %d, want 30", FG_BLACK)
	}
	if FG_BRIGHT_BLACK != 90 {
		t.Fatalf("FG_BRIGHT_BLACK = %d, want 90", FG_BRIGHT_BLACK)
	}
	if BG_BLACK != 40 {
		t.Fatalf("BG_BLACK = %d, want 40", BG_BLACK)
	}
	if BG_BRIGHT_BLACK != 100 {
		t.Fatalf("BG_BRIGHT_BLACK = %d, want 100", BG_BRIGHT_BLACK)
	}
}

func TestNewRGB(t *testing.T) {
	const r, g, b = uint8(0x12), uint8(0x34), uint8(0x56)

	gotR, gotG, gotB, fg, ok := NewRGB(r, g, b).isRGB()
	if !ok || !fg {
		t.Fatalf("NewRGB 应识别为前景真彩色: ok=%v fg=%v", ok, fg)
	}
	if gotR != r || gotG != g || gotB != b {
		t.Fatalf("NewRGB 分量 = (%d,%d,%d), want (%d,%d,%d)", gotR, gotG, gotB, r, g, b)
	}

	gotR, gotG, gotB, fg, ok = NewBgRGB(r, g, b).isRGB()
	if !ok || fg {
		t.Fatalf("NewBgRGB 应识别为背景真彩色: ok=%v fg=%v", ok, fg)
	}
	if gotR != r || gotG != g || gotB != b {
		t.Fatalf("NewBgRGB 分量 = (%d,%d,%d), want (%d,%d,%d)", gotR, gotG, gotB, r, g, b)
	}
}

func TestIsRGBFalseForBasicCode(t *testing.T) {
	for _, code := range []Code{STYLE_BOLD, FG_RED, FG_BRIGHT_WHITE, BG_BLUE, BG_BRIGHT_WHITE} {
		if _, _, _, _, ok := code.isRGB(); ok {
			t.Fatalf("基础代码 %d 不应被识别为真彩色", int(code))
		}
	}
}

func TestNewRGBDistinguishesForegroundAndBackground(t *testing.T) {
	if NewRGB(1, 2, 3) == NewBgRGB(1, 2, 3) {
		t.Fatal("前景真彩色与背景真彩色代码不应相等")
	}
	if NewRGB(1, 2, 3) == NewRGB(1, 2, 4) {
		t.Fatal("不同分量的真彩色代码不应相等")
	}
}
