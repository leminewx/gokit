package mesh

import (
	"testing"
	"time"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.Debug != false {
		t.Error("Debug should default to false")
	}
	if cfg.MessageTTL != 0 {
		t.Error("MessageTTL should default to 0")
	}
	if cfg.MessageSendChanSize != 128 {
		t.Errorf("MessageSendChanSize = %d, want 128", cfg.MessageSendChanSize)
	}
	if cfg.MaxSeenAge != 1*time.Minute {
		t.Errorf("MaxSeenAge = %v, want 1m", cfg.MaxSeenAge)
	}
	if cfg.RequestTimeout != 5*time.Second {
		t.Errorf("RequestTimeout = %v, want 5s", cfg.RequestTimeout)
	}
	if cfg.ConnDialTimeout != 5*time.Second {
		t.Errorf("ConnDialTimeout = %v, want 5s", cfg.ConnDialTimeout)
	}
	if cfg.GossipInterval != 2*time.Second {
		t.Errorf("GossipInterval = %v, want 2s", cfg.GossipInterval)
	}
	if cfg.GossipFanoutSize != 3 {
		t.Errorf("GossipFanoutSize = %d, want 3", cfg.GossipFanoutSize)
	}
	if cfg.GossipPartialViewSize != 20 {
		t.Errorf("GossipPartialViewSize = %d, want 20", cfg.GossipPartialViewSize)
	}
	if cfg.GossipFailureTimeout != 15*time.Second {
		t.Errorf("GossipFailureTimeout = %v, want 15s", cfg.GossipFailureTimeout)
	}
}

func TestOption_WithDebug(t *testing.T) {
	cfg := NewDefaultConfig()
	WithDebug()(cfg)
	if !cfg.Debug {
		t.Error("Debug should be true after WithDebug")
	}
}

func TestOption_WithMessageTTL(t *testing.T) {
	cfg := NewDefaultConfig()

	// positive value
	WithMessageTTL(5)(cfg)
	if cfg.MessageTTL != 5 {
		t.Errorf("MessageTTL = %d, want 5", cfg.MessageTTL)
	}

	// zero or negative should be ignored
	WithMessageTTL(0)(cfg)
	if cfg.MessageTTL != 5 {
		t.Error("MessageTTL should not change for value <= 0")
	}
	WithMessageTTL(-1)(cfg)
	if cfg.MessageTTL != 5 {
		t.Error("MessageTTL should not change for negative value")
	}
}

func TestOption_WithMessageSendChanSize(t *testing.T) {
	cfg := NewDefaultConfig()

	WithMessageSendChanSize(256)(cfg)
	if cfg.MessageSendChanSize != 256 {
		t.Errorf("MessageSendChanSize = %d, want 256", cfg.MessageSendChanSize)
	}

	WithMessageSendChanSize(0)(cfg)
	if cfg.MessageSendChanSize != 256 {
		t.Error("MessageSendChanSize should not change for zero value")
	}
}

func TestOption_WithMaxSeenAge(t *testing.T) {
	cfg := NewDefaultConfig()
	WithMaxSeenAge(30 * time.Second)(cfg)
	if cfg.MaxSeenAge != 30*time.Second {
		t.Errorf("MaxSeenAge = %v, want 30s", cfg.MaxSeenAge)
	}

	WithMaxSeenAge(0)(cfg)
	if cfg.MaxSeenAge != 30*time.Second {
		t.Error("MaxSeenAge should not change for zero value")
	}
}

func TestOption_WithRequestTimeout(t *testing.T) {
	cfg := NewDefaultConfig()
	WithRequestTimeout(10 * time.Second)(cfg)
	if cfg.RequestTimeout != 10*time.Second {
		t.Errorf("RequestTimeout = %v, want 10s", cfg.RequestTimeout)
	}
}

func TestOption_WithConnDialTimeout(t *testing.T) {
	cfg := NewDefaultConfig()
	WithConnDialTimeout(3 * time.Second)(cfg)
	if cfg.ConnDialTimeout != 3*time.Second {
		t.Errorf("ConnDialTimeout = %v, want 3s", cfg.ConnDialTimeout)
	}
}

func TestOption_WithGossipInterval(t *testing.T) {
	cfg := NewDefaultConfig()
	WithGossipInterval(5 * time.Second)(cfg)
	if cfg.GossipInterval != 5*time.Second {
		t.Errorf("GossipInterval = %v, want 5s", cfg.GossipInterval)
	}
}

func TestOption_WithGossipFanoutSize(t *testing.T) {
	cfg := NewDefaultConfig()

	WithGossipFanoutSize(5)(cfg)
	if cfg.GossipFanoutSize != 5 {
		t.Errorf("GossipFanoutSize = %d, want 5", cfg.GossipFanoutSize)
	}

	WithGossipFanoutSize(0)(cfg)
	if cfg.GossipFanoutSize != 5 {
		t.Error("GossipFanoutSize should not change for zero value")
	}
}

func TestOption_WithGossipPartialViewSize(t *testing.T) {
	cfg := NewDefaultConfig()

	WithGossipPartialViewSize(30)(cfg)
	if cfg.GossipPartialViewSize != 30 {
		t.Errorf("GossipPartialViewSize = %d, want 30", cfg.GossipPartialViewSize)
	}

	WithGossipPartialViewSize(0)(cfg)
	if cfg.GossipPartialViewSize != 30 {
		t.Error("GossipPartialViewSize should not change for zero value")
	}
}

func TestOption_WithGossipFailureTimeout(t *testing.T) {
	cfg := NewDefaultConfig()
	WithGossipFailureTimeout(30 * time.Second)(cfg)
	if cfg.GossipFailureTimeout != 30*time.Second {
		t.Errorf("GossipFailureTimeout = %v, want 30s", cfg.GossipFailureTimeout)
	}
}

func TestOption_Chain(t *testing.T) {
	cfg := NewDefaultConfig()
	WithDebug()(cfg)
	WithMessageTTL(3)(cfg)
	WithGossipFanoutSize(5)(cfg)
	WithRequestTimeout(10 * time.Second)(cfg)

	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if cfg.MessageTTL != 3 {
		t.Errorf("MessageTTL = %d, want 3", cfg.MessageTTL)
	}
	if cfg.GossipFanoutSize != 5 {
		t.Errorf("GossipFanoutSize = %d, want 5", cfg.GossipFanoutSize)
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Errorf("RequestTimeout = %v, want 10s", cfg.RequestTimeout)
	}
}

func TestConfig_LogDebug(t *testing.T) {
	// 关闭调试模式时应无输出
	cfg := NewDefaultConfig()
	cfg.logDebug("should not appear") // 不应panic

	// 开启调试模式
	WithDebug()(cfg)
	cfg.logDebug("debug msg: %d", 123) // 手动验证控制台输出
}
