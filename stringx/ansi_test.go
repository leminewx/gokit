package stringx

import (
	"bytes"
	"testing"
)

func TestStripANSIString(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                          "",
		"plain text":                "plain text",
		"\x1B[31mred\x1B[0m":        "red",
		"\x1B[1;32mgreen\x1B[0m ok": "green ok",
		"\x1B[2Jclear screen":       "clear screen",
		"\x1Bno-bracket":            "\x1Bno-bracket",
	}

	for input, want := range cases {
		if got := StripANSIString(input); got != want {
			t.Errorf("StripANSIString(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStripANSIBytes(t *testing.T) {
	t.Parallel()

	if got := StripANSIBytes(nil); got != nil {
		t.Errorf("StripANSIBytes(nil) = %v, want nil", got)
	}

	if got := StripANSIBytes([]byte("plain")); !bytes.Equal(got, []byte("plain")) {
		t.Errorf("StripANSIBytes(plain) = %q, want %q", got, "plain")
	}

	got := StripANSIBytes([]byte("\x1B[31mred\x1B[0m"))
	if !bytes.Equal(got, []byte("red")) {
		t.Errorf("StripANSIBytes(带转义序列) = %q, want %q", got, "red")
	}

	// 仅含 ESC 而无 CSI 结构时保持原样
	raw := []byte("\x1Bno-bracket")
	if got := StripANSIBytes(raw); !bytes.Equal(got, raw) {
		t.Errorf("StripANSIBytes(孤立 ESC) = %q, want %q", got, raw)
	}
}

func BenchmarkStripANSIString(b *testing.B) {
	input := "\x1B[1;32m[INFO]\x1B[0m server started on :8080"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = StripANSIString(input)
	}
}
