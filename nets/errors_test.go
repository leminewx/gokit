package nets

import (
	"strings"
	"testing"
)

// TestErrorMessages 校验本包错误哨兵的消息遵循统一规范：
// 统一前缀、详情首字母小写、详情不以句号结尾。
func TestErrorMessages(t *testing.T) {
	t.Parallel()

	cases := []error{
		ErrInvalidCIDR,
		ErrIPv4Only,
		ErrInvalidMask,
		ErrNoValidIP,
		ErrInvalidPreference,
		ErrTooManyAddresses,
	}

	for _, err := range cases {
		message := err.Error()
		if !strings.HasPrefix(message, _ERROR_PREFIX) {
			t.Errorf("%v: 缺少统一前缀 %q", err, _ERROR_PREFIX)
			continue
		}

		detail := strings.TrimPrefix(message, _ERROR_PREFIX)
		if detail == "" {
			t.Errorf("%v: 错误详情为空", err)
			continue
		}

		if detail[0] >= 'A' && detail[0] <= 'Z' {
			t.Errorf("%v: 错误详情首字母必须小写", err)
		}

		if strings.HasSuffix(detail, ".") {
			t.Errorf("%v: 错误详情不得以句号结尾", err)
		}
	}
}
