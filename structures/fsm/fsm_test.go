package fsm

import (
	"errors"
	"fmt"
	"testing"
)

func TestFSM(t *testing.T) {
	// 定义自定义数据类型，包含设备信息和额外上下文
	type EventData struct {
		UserID   string
		TaskID   string
		DeviceID string
	}

	// 创建泛型 FSM，指定 data 类型为 EventData
	sm := New(
		WithStopOnError[EventData](true),
	)

	// 添加状态
	sm.AddStates("ready", "locked", "running")

	// 添加转移规则
	sm.AddTransition("ready", "lock", "locked")
	sm.AddTransition("locked", "start", "running")

	// 注册事件回调：使用类型安全的 data 参数
	sm.CallOnEvent("lock", func(from, to State, data EventData) error {
		fmt.Printf("Locking device %s by user %s\n", data.DeviceID, data.UserID)
		return nil
	})

	// 触发转移
	newState, err := sm.Trigger("ready", "lock", EventData{
		DeviceID: "V2X-001",
		UserID:   "tester1",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("New state:", newState)
}

// TestFSM_UpsertTransition 验证 UpsertTransition 幂等覆盖语义
//   - 新名字语义与实现一致（旧名 MustAddTransition 失败并不 panic）
func TestFSM_UpsertTransition(t *testing.T) {
	sm := New[struct{}]()
	sm.AddStates("a", "b", "c")

	if err := sm.UpsertTransition("a", "go", "b"); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	// 重复 upsert 应覆盖而不是报 ErrTransitionExists
	if err := sm.UpsertTransition("a", "go", "c"); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	to, err := sm.Trigger("a", "go", struct{}{})
	if err != nil || to != "c" {
		t.Fatalf("trigger = %q, %v; want c, nil", to, err)
	}

	// 旧名必须保持兼容
	if err := sm.MustAddTransition("a", "go", "b"); err != nil {
		t.Fatalf("deprecated MustAddTransition: %v", err)
	}
	if to, err := sm.Trigger("a", "go", struct{}{}); err != nil || to != "b" {
		t.Fatalf("trigger after MustAddTransition = %q, %v; want b, nil", to, err)
	}

	// 状态不存在时仍返回错误
	if err := sm.UpsertTransition("a", "go", "missing"); err == nil {
		t.Fatal("unknown target state should return error")
	}
}

// TestFSM_TransitionLifecycle 覆盖规则校验、三类回调执行顺序与最终状态
func TestFSM_TransitionLifecycle(t *testing.T) {
	sm := New[string]()
	sm.AddStates("idle", "running", "stopped")

	// 未注册状态必须报错
	if err := sm.AddTransition("idle", "go", "unknown"); err == nil {
		t.Fatal("unknown target state should fail")
	}
	if err := sm.AddTransition("unknown", "go", "idle"); err == nil {
		t.Fatal("unknown source state should fail")
	}

	if err := sm.AddTransition("idle", "start", "running"); err != nil {
		t.Fatalf("add transition: %v", err)
	}
	// 同一 (from, event) 重复注册必须报错
	if err := sm.AddTransition("idle", "start", "stopped"); !errors.Is(err, ErrTransitionExists) {
		t.Fatalf("duplicate transition err = %v, want ErrTransitionExists", err)
	}

	var order []string
	sm.CallOnEvent("start", func(from, to State, data string) error {
		order = append(order, "event:"+string(from)+"->"+string(to))
		return nil
	})
	if err := sm.CallOnLeave("idle", func(state State, data string) error {
		order = append(order, "leave:"+string(state))
		return nil
	}); err != nil {
		t.Fatalf("CallOnLeave: %v", err)
	}
	if err := sm.CallOnEnter("running", func(state State, data string) error {
		order = append(order, "enter:"+string(state))
		return nil
	}); err != nil {
		t.Fatalf("CallOnEnter: %v", err)
	}

	to, err := sm.Trigger("idle", "start", "payload")
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if to != "running" {
		t.Fatalf("state = %q, want running", to)
	}

	want := []string{"event:idle->running", "leave:idle", "enter:running"}
	if len(order) != len(want) {
		t.Fatalf("callback order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("callback order = %v, want %v", order, want)
		}
	}

	// 未注册的事件
	if _, err := sm.Trigger("idle", "stop", ""); !errors.Is(err, ErrTransitionNotFound) {
		t.Fatalf("unknown event err = %v, want ErrTransitionNotFound", err)
	}

	// 回调注册到未声明状态同样报错
	if err := sm.CallOnLeave("ghost", nil); err == nil {
		t.Fatal("CallOnLeave on unknown state should fail")
	}
	if err := sm.CallOnEnter("ghost", nil); err == nil {
		t.Fatal("CallOnEnter on unknown state should fail")
	}
}

// TestFSM_CallbackErrorPolicy 覆盖回调出错时中止/放行两种策略
func TestFSM_CallbackErrorPolicy(t *testing.T) {
	errBoom := errors.New("boom")

	build := func(stop bool) *FSM[struct{}] {
		sm := New[struct{}](WithStopOnError[struct{}](stop))
		sm.AddStates("a", "b")
		_ = sm.AddTransition("a", "go", "b")
		return sm
	}

	// stop=true：事件回调失败后不执行状态转移
	sm := build(true)
	sm.CallOnEvent("go", func(_, _ State, _ struct{}) error { return errBoom })
	if to, err := sm.Trigger("a", "go", struct{}{}); !errors.Is(err, ErrEventCallback) || to != "a" {
		t.Fatalf("stop policy: to = %q, err = %v", to, err)
	}

	// stop=false：回调失败仍完成转移
	sm = build(false)
	sm.CallOnEvent("go", func(_, _ State, _ struct{}) error { return errBoom })
	sm.CallOnLeave("a", func(_ State, _ struct{}) error { return errBoom })
	sm.CallOnEnter("b", func(_ State, _ struct{}) error { return errBoom })
	if to, err := sm.Trigger("a", "go", struct{}{}); err != nil || to != "b" {
		t.Fatalf("skip policy: to = %q, err = %v", to, err)
	}

	// 单独策略：仅离开回调中止（事件/进入策略已关闭）
	strict := New[struct{}](WithLeaveErrorPolicy[struct{}](true), WithEventErrorPolicy[struct{}](false), WithEnterErrorPolicy[struct{}](false))
	strict.AddStates("a", "b")
	_ = strict.AddTransition("a", "go", "b")
	strict.CallOnLeave("a", func(_ State, _ struct{}) error { return errBoom })
	if to, err := strict.Trigger("a", "go", struct{}{}); !errors.Is(err, ErrLeaveCallback) || to != "a" {
		t.Fatalf("leave policy: to = %q, err = %v", to, err)
	}
}

// TestFSM_ExportImport 覆盖配置导出/导入与校验
func TestFSM_ExportImport(t *testing.T) {
	sm := New[struct{}]()
	sm.AddStates("a", "b", "c")
	_ = sm.AddTransition("a", "go", "b")
	_ = sm.UpsertTransition("b", "back", "a")

	states, transitions := sm.Export()
	if len(states) != 3 || transitions["a"]["go"] != "b" || transitions["b"]["back"] != "a" {
		t.Fatalf("export = %v, %v", states, transitions)
	}

	// GetStates 返回的是副本，修改不影响内部
	states[0] = "hacked"
	if sm.HasState("hacked") {
		t.Fatal("GetStates should return a copy")
	}

	other := New[struct{}]()
	// 缺少转移规则引用的状态，导入必须失败
	if err := other.Import([]State{"b", "c"}, transitions); err == nil {
		t.Fatal("import with unknown state should fail")
	}

	if err := other.Import([]State{"a", "b", "c"}, transitions); err != nil {
		t.Fatalf("import: %v", err)
	}
	// 导入后回调被清空，但规则可用
	if to, err := other.Trigger("a", "go", struct{}{}); err != nil || to != "b" {
		t.Fatalf("trigger after import: %q, %v", to, err)
	}

	stats := other.GetStats()
	if stats["states"] != 3 {
		t.Fatalf("stats = %v", stats)
	}
	if err := other.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// 导入非法规则：目标状态不存在
	bad := New[struct{}]()
	if err := bad.Import([]State{"a"}, map[State]map[Event]State{"a": {"go": "ghost"}}); err == nil {
		t.Fatal("import with unknown target should fail")
	}
	// 导入非法规则：来源状态不存在
	if err := bad.Import([]State{"a"}, map[State]map[Event]State{"ghost": {"go": "a"}}); err == nil {
		t.Fatal("import with unknown source should fail")
	}
}

// TestFSM_RemoveTransition 覆盖转移规则删除
func TestFSM_RemoveTransition(t *testing.T) {
	sm := New[struct{}]()
	sm.AddStates("a", "b")
	_ = sm.AddTransition("a", "go", "b")

	// 不存在的来源状态是幂等的
	if err := sm.RemoveTransition("b", "go"); err != nil {
		t.Fatalf("remove from state without rules: %v", err)
	}
	// 存在来源但事件不存在时报错
	if err := sm.RemoveTransition("a", "stop"); !errors.Is(err, ErrTransitionNotFound) {
		t.Fatalf("remove unknown event err = %v", err)
	}
	if err := sm.RemoveTransition("a", "go"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, ok := sm.GetTransitions()["a"]; ok {
		t.Fatal("empty rules should remove the from-state key")
	}
}
