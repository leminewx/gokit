package stringx

import (
	"strings"
	"testing"
)

func TestConvertToSnakeCase(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":             "",
		"HelloWorld":   "hello_world",
		"HTTPServer":   "http_server",
		"UserID":       "user_id",
		"ABC":          "abc",
		"With2Number":  "with2_number",
		"already_snak": "already_snak",
	}

	for input, want := range cases {
		if got := ConvertToSnakeCase(input); got != want {
			t.Errorf("ConvertToSnakeCase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestHumanizeBytes(t *testing.T) {
	t.Parallel()

	cases := map[int64]string{
		0:       "0 B",
		1023:    "1023 B",
		1 << 10: "1.00 KiB",
		1 << 20: "1.00 MiB",
		1 << 30: "1.00 GiB",
		1 << 40: "1.00 TiB",
		1 << 50: "1.00 PiB",
	}

	for bytes, want := range cases {
		if got := HumanizeBytes(bytes); got != want {
			t.Errorf("HumanizeBytes(%d) = %q, want %q", bytes, got, want)
		}
	}

	// 负数不做单位换算，按字节原值输出（带负号）
	if got, want := HumanizeBytes(-1), "-1 B"; got != want {
		t.Errorf("HumanizeBytes(-1) = %q, want %q", got, want)
	}
	if got := HumanizeBytes(-(1 << 20)); !strings.HasPrefix(got, "-") {
		t.Errorf("HumanizeBytes(负数) = %q, want 负号前缀", got)
	}
}

func BenchmarkConvertToSnakeCase(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ConvertToSnakeCase("HTTPServerRequestID")
	}
}
