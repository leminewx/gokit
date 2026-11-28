package serial

import (
	"errors"
	"testing"
	"time"
)

func TestConfigNormalize(t *testing.T) {
	t.Parallel()

	config := Config{Name: "  /dev/ttyS0  ", Parity: 'o'}.Normalize()

	if config.Name != "/dev/ttyS0" {
		t.Fatalf("Name = %q, want %q", config.Name, "/dev/ttyS0")
	}
	if config.Baud != _DEFAULT_BAUD {
		t.Errorf("Baud = %d, want %d", config.Baud, _DEFAULT_BAUD)
	}
	if config.DataBits != _DEFAULT_DATA_BITS {
		t.Errorf("DataBits = %d, want %d", config.DataBits, _DEFAULT_DATA_BITS)
	}
	if config.StopBits != _DEFAULT_STOP_BITS {
		t.Errorf("StopBits = %d, want %d", config.StopBits, _DEFAULT_STOP_BITS)
	}
	if config.Parity != 'O' {
		t.Errorf("Parity = %q, want 'O'", config.Parity)
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	valid := Config{Name: "COM1"}.Normalize()

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "空名称", mutate: func(c *Config) { c.Name = "" }},
		{name: "波特率为零", mutate: func(c *Config) { c.Baud = 0 }},
		{name: "数据位非法", mutate: func(c *Config) { c.DataBits = 9 }},
		{name: "停止位非法", mutate: func(c *Config) { c.StopBits = 3 }},
		{name: "校验位非法", mutate: func(c *Config) { c.Parity = 'X' }},
		{name: "流控非法", mutate: func(c *Config) { c.FlowControl = FlowControlType(99) }},
		{name: "读超时为负", mutate: func(c *Config) { c.ReadTimeout = -time.Second }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			config := valid
			test.mutate(&config)

			if err := config.Validate(); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("Validate() = %v, want ErrInvalidConfig", err)
			}
		})
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("有效配置校验失败: %v", err)
	}
}

func TestFlowControlTypeString(t *testing.T) {
	t.Parallel()

	tests := map[FlowControlType]string{
		FLOW_CONTROL_NONE:     "none",
		FLOW_CONTROL_RTS_CTS:  "rts-cts",
		FLOW_CONTROL_DTR_DSR:  "dtr-dsr",
		FLOW_CONTROL_XON_XOFF: "xon-xoff",
		FlowControlType(99):   "unknown",
	}

	for flow, want := range tests {
		if got := flow.String(); got != want {
			t.Errorf("FlowControlType(%d).String() = %q, want %q", int(flow), got, want)
		}
	}
}

func TestOpenRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	if _, err := Open(Config{}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Open(空配置) = %v, want ErrInvalidConfig", err)
	}
}
