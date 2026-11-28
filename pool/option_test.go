package pool

import "testing"

func TestOptions(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	WithWorkers(4)(cfg)
	WithQueueSize(16)(cfg)

	var handled any
	WithPanicHandler(func(recv any, _ []byte) { handled = recv })(cfg)

	if cfg.workers != 4 {
		t.Errorf("workers = %d, want 4", cfg.workers)
	}
	if cfg.queueSize != 16 {
		t.Errorf("queueSize = %d, want 16", cfg.queueSize)
	}
	if cfg.panicHandler == nil {
		t.Fatal("panicHandler 未设置")
	}
	cfg.panicHandler("boom", nil)
	if handled != "boom" {
		t.Errorf("panicHandler 收到 %v, want boom", handled)
	}
}

func TestNewAppliesOptions(t *testing.T) {
	t.Parallel()

	instance := New(WithWorkers(2), WithQueueSize(4))
	defer instance.Release()

	if got := instance.GetWorkers(); got != 2 {
		t.Errorf("GetWorkers() = %d, want 2", got)
	}
}

func TestNewDefaults(t *testing.T) {
	t.Parallel()

	instance := New()
	defer instance.Release()

	if got := instance.GetWorkers(); got <= 0 {
		t.Errorf("默认 GetWorkers() = %d, want > 0", got)
	}
}
