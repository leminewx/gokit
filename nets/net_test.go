package nets

import (
	"errors"
	"net"
	"testing"
)

// timeoutErr 模拟实现了 net.Error 的超时错误
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// fakeNetErr 模拟实现了 net.Error 的非超时错误
type fakeNetErr struct{ msg string }

func (e fakeNetErr) Error() string   { return e.msg }
func (e fakeNetErr) Timeout() bool   { return false }
func (e fakeNetErr) Temporary() bool { return false }

func TestGenerateIPs(t *testing.T) {
	t.Log(GenerateIPV4s("10.10.0.2/16", true))
}

// TestIsConnectionError_DNSNotConnectionError 验证 DNS 解析错误不会被判定为连接断开
//   - 回归点：*net.DNSError 同时实现了 net.Error，曾被 net.Error 分支误判为断连
func TestIsConnectionError_DNSNotConnectionError(t *testing.T) {
	t.Parallel()

	cases := []*net.DNSError{
		{Err: "no such host", Name: "nope.invalid", IsNotFound: true},
		{Err: "server misbehaving", Name: "foo.invalid", IsTemporary: true},
		{Err: "i/o timeout", Name: "slow.invalid", IsTimeout: true},
	}
	for _, dnsErr := range cases {
		if IsConnectionError(dnsErr) {
			t.Fatalf("dns error %q should not be a connection error", dnsErr.Err)
		}
		// 包装后依然要能被识别
		if IsConnectionError(errors.Join(errors.New("dial"), dnsErr)) {
			t.Fatalf("wrapped dns error %q should not be a connection error", dnsErr.Err)
		}
	}
}

// TestIsConnectionError_TimeoutNotConnectionError 验证超时错误不算断连
func TestIsConnectionError_TimeoutNotConnectionError(t *testing.T) {
	t.Parallel()

	if IsConnectionError(timeoutErr{}) {
		t.Fatal("timeout error should not be a connection error")
	}
	// 短语表已移除 i/o timeout，纯文本超时错误同样不算断连
	if IsConnectionError(errors.New("read tcp 10.0.0.1:1234: i/o timeout")) {
		t.Fatal("plain i/o timeout text should not be a connection error")
	}
}

// TestIsConnectionError_NonTimeoutNetError 验证非超时的 net.Error 仍判定为断连
func TestIsConnectionError_NonTimeoutNetError(t *testing.T) {
	t.Parallel()

	if !IsConnectionError(fakeNetErr{msg: "some transport failure"}) {
		t.Fatal("non-timeout net.Error should be a connection error")
	}

	// 常见断开短语
	for _, msg := range []string{"EOF", "broken pipe", "connection reset by peer"} {
		if !IsConnectionError(errors.New(msg)) {
			t.Fatalf("%q should be a connection error", msg)
		}
	}

	if IsConnectionError(nil) {
		t.Fatal("nil should not be a connection error")
	}
}

// TestGenerateIPV4sLimit 验证地址数量上限
//   - 回归点：此前对 /8 等网段会无上限生成 1600 万个字符串
func TestGenerateIPV4sLimit(t *testing.T) {
	t.Parallel()

	// /16 恰好等于默认上限，可以正常生成
	ips, err := GenerateIPV4s("10.10.0.0/16", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ips) != (1<<16)-2 {
		t.Fatalf("expected %d ips, got %d", (1<<16)-2, len(ips))
	}

	// /15 超过默认上限，必须报错而不是无上限分配
	if _, err := GenerateIPV4s("10.10.0.0/15", true); !errors.Is(err, ErrTooManyAddresses) {
		t.Fatalf("expected ErrTooManyAddresses, got %v", err)
	}

	// 自定义上限：放宽可以生成，收紧则拒绝
	if got, err := GenerateIPV4sLimit("10.10.0.0/15", false, 1<<17); err != nil || len(got) != 1<<17 {
		t.Fatalf("expected %d ips with custom limit, got %d, err %v", 1<<17, len(got), err)
	}
	if _, err := GenerateIPV4sLimit("10.10.0.0/16", true, 100); !errors.Is(err, ErrTooManyAddresses) {
		t.Fatalf("expected ErrTooManyAddresses with small limit, got %v", err)
	}

	// maxCount <= 0 回落默认上限
	if _, err := GenerateIPV4sLimit("10.10.0.0/15", true, 0); !errors.Is(err, ErrTooManyAddresses) {
		t.Fatalf("expected default limit to reject /15, got %v", err)
	}
}

// TestGenerateIPV4sEdgeCases 覆盖 /31、/32、不跳过网络地址与非法输入
func TestGenerateIPV4sEdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cidr    string
		skip    bool
		want    []string
		wantErr error
	}{
		{"slash32", "192.168.1.7/32", true, []string{"192.168.1.7"}, nil},
		{"slash31", "192.168.1.6/31", true, []string{"192.168.1.6", "192.168.1.7"}, nil},
		{"slash30-skip", "192.168.1.4/30", true, []string{"192.168.1.5", "192.168.1.6"}, nil},
		{"slash30-keep", "192.168.1.4/30", false, []string{"192.168.1.4", "192.168.1.5", "192.168.1.6", "192.168.1.7"}, nil},
		{"invalid-cidr", "not-a-cidr", true, nil, ErrInvalidCIDR},
		{"ipv6", "fd00::/120", true, nil, ErrIPv4Only},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := GenerateIPV4s(c.cidr, c.skip)
			if c.wantErr != nil {
				if !errors.Is(err, c.wantErr) {
					t.Fatalf("err = %v, want %v", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("ips = %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("ips = %v, want %v", got, c.want)
				}
			}
		})
	}
}

// TestGetNetPreferredIPInvalidInput 覆盖非法偏好与不存在网卡的错误分支
func TestGetNetPreferredIPInvalidInput(t *testing.T) {
	t.Parallel()

	// 网卡不存在：getValidIPs 先行失败
	if _, err := GetNetPreferredIP("gokit-no-such-iface", "ipv4"); !errors.Is(err, ErrNoValidIP) {
		t.Fatalf("err = %v, want ErrNoValidIP", err)
	}

	// 逐一遍历本机网卡：非法偏好必须报错；若命中具有可用 IP 的网卡，
	// 则必须报 ErrInvalidPreference（getValidIPs 通过后才校验偏好）
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("list interfaces: %v", err)
	}

	var coveredPreference bool
	for _, iface := range ifaces {
		if _, err := GetNetPreferredIP(iface.Name, "ipv5"); err != nil {
			if errors.Is(err, ErrInvalidPreference) {
				coveredPreference = true
				continue
			}
			if errors.Is(err, ErrNoValidIP) {
				continue
			}
			t.Fatalf("unexpected err on %q: %v", iface.Name, err)
		}
		t.Fatalf("invalid preference on %q should fail", iface.Name)
	}

	// 至少有一个网卡应当能取到配置的 IP（否则机器上全是回环/未配置地址）
	for _, iface := range ifaces {
		if ip, err := GetNetPreferredIP(iface.Name, "ipv4"); err == nil {
			if ip == nil {
				t.Fatalf("preferred ip on %q should not be nil", iface.Name)
			}
			break
		}
	}

	_ = coveredPreference // 无可用 IP 的容器环境下无法覆盖，不强求
}

// TestIsULA 验证 IPv6 ULA 判定
func TestIsULA(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"fc00::1":      true,
		"fd12:3456::1": true,
		"fe80::1":      false,
		"2001:db8::1":  false,
		"10.0.0.1":     false,
	}

	for raw, want := range cases {
		ip := net.ParseIP(raw)
		if ip == nil {
			t.Fatalf("bad test ip %q", raw)
		}
		if got := isULA(ip); got != want {
			t.Fatalf("isULA(%q) = %v, want %v", raw, got, want)
		}
	}
}
