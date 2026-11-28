package fsm

import (
	"sync"
	"sync/atomic"
)

// ----------------------------------------------------------- 类型定义 -----------------------------------------------------------

type (
	// State 状态类型
	State string

	// Event 事件类型
	Event string

	// OnLeaveHandler 状态离开时的回调函数
	OnLeaveHandler[T any] func(state State, data T) error

	// OnEnterHandler 状态进入时的回调函数
	OnEnterHandler[T any] func(state State, data T) error

	// OnEventHandler 状态转移时的回调函数
	OnEventHandler[T any] func(from, to State, data T) error
)

// ----------------------------------------------------------- 转移规则 -----------------------------------------------------------

// rule 转移规则
type rule struct {
	to    State
	event Event
}

// ----------------------------------------------------------- 核心结构 -----------------------------------------------------------

// FSM 有限状态机的结构类型
type FSM[T any] struct {
	// 状态集合
	states map[State]struct{}

	// 转移规则：from -> []rule
	transitions map[State][]rule

	// 回调
	onEvent map[Event][]OnEventHandler[T]
	onLeave map[State][]OnLeaveHandler[T]
	onEnter map[State][]OnEnterHandler[T]

	// 运行时选项
	stopOnEventError bool
	stopOnLeaveError bool
	stopOnEnterError bool

	// 统计
	callCount uint64

	// 配置锁（保护配置读写，通常初始化完成后很少变更）
	lock sync.RWMutex
}

// New 新建一个有限状态机实例
func New[T any](opts ...Option[T]) *FSM[T] {
	f := &FSM[T]{
		states:           make(map[State]struct{}),
		transitions:      make(map[State][]rule),
		onEvent:          make(map[Event][]OnEventHandler[T]),
		onLeave:          make(map[State][]OnLeaveHandler[T]),
		onEnter:          make(map[State][]OnEnterHandler[T]),
		stopOnEventError: true,
		stopOnLeaveError: true,
		stopOnEnterError: true,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// ------------------------------------------------------ 状态管理 ------------------------------------------------------

// AddStates 添加状态到集合
func (own *FSM[T]) AddStates(states ...State) {
	own.lock.Lock()
	for _, state := range states {
		own.states[state] = struct{}{}
	}
	own.lock.Unlock()
}

// HasState 判断状态集合中是否存在指定的状态
func (own *FSM[T]) HasState(state State) bool {
	own.lock.RLock()
	_, ok := own.states[state]
	own.lock.RUnlock()
	return ok
}

// ------------------------------------------------------ 规则管理 ------------------------------------------------------

// AddTransition 添加转移规则，若已存在相同 (from, event) 则返回错误
func (own *FSM[T]) AddTransition(from State, event Event, to State) error {
	own.lock.Lock()
	defer own.lock.Unlock()

	if _, ok := own.states[from]; !ok {
		return wrapStateError(from)
	}

	if _, ok := own.states[to]; !ok {
		return wrapStateError(to)
	}

	for _, rule := range own.transitions[from] {
		if rule.event == event {
			return wrapError(ErrTransitionExists, "from=%s, event=%s, to=%s", from, event, to)
		}
	}

	own.transitions[from] = append(own.transitions[from], rule{event: event, to: to})
	return nil
}

// UpsertTransition 添加转移规则，若已存在相同 (from, event) 则覆盖目标状态
//   - 幂等操作：重复调用不会返回“规则已存在”错误，适合配置热更新场景
func (own *FSM[T]) UpsertTransition(from State, event Event, to State) error {
	own.lock.Lock()
	defer own.lock.Unlock()

	if _, ok := own.states[from]; !ok {
		return wrapStateError(from)
	}

	if _, ok := own.states[to]; !ok {
		return wrapStateError(to)
	}

	rules := own.transitions[from]
	for i, rule := range rules {
		if rule.event == event {
			rules[i].to = to
			return nil
		}
	}

	own.transitions[from] = append(rules, rule{event: event, to: to})
	return nil
}

// MustAddTransition 添加转移规则，若已存在则覆盖
//
// Deprecated: 该名称容易让人以为“失败会 panic”，实际只返回 error，
// 请使用语义准确的 UpsertTransition。
func (own *FSM[T]) MustAddTransition(from State, event Event, to State) error {
	return own.UpsertTransition(from, event, to)
}

// RemoveTransition 删除转移规则
func (own *FSM[T]) RemoveTransition(from State, event Event) error {
	own.lock.Lock()
	defer own.lock.Unlock()

	rules, ok := own.transitions[from]
	if !ok {
		return nil
	}

	for i, rule := range rules {
		if rule.event == event {
			own.transitions[from] = append(rules[:i], rules[i+1:]...)
			if len(own.transitions[from]) == 0 {
				delete(own.transitions, from)
			}
			return nil
		}
	}

	return wrapError(ErrTransitionNotFound, "from=%s, event=%s", from, event)
}

// ------------------------------------------------------ 回调注册 ------------------------------------------------------

// CallOnEvent 添加处理事件时的回调函数
func (own *FSM[T]) CallOnEvent(event Event, handlers ...OnEventHandler[T]) {
	own.lock.Lock()
	own.onEvent[event] = append(own.onEvent[event], handlers...)
	own.lock.Unlock()
}

// CallOnLeave 添加状态离开时的回调函数
func (own *FSM[T]) CallOnLeave(state State, handlers ...OnLeaveHandler[T]) error {
	own.lock.Lock()
	defer own.lock.Unlock()

	if _, ok := own.states[state]; !ok {
		return wrapStateError(state)
	}

	own.onLeave[state] = append(own.onLeave[state], handlers...)
	return nil
}

// CallOnEnter 添加状态进入时的回调函数
func (own *FSM[T]) CallOnEnter(state State, handlers ...OnEnterHandler[T]) error {
	own.lock.Lock()
	defer own.lock.Unlock()

	if _, ok := own.states[state]; !ok {
		return wrapStateError(state)
	}

	own.onEnter[state] = append(own.onEnter[state], handlers...)
	return nil
}

// ------------------------------------------------------ 状态转移 ------------------------------------------------------

// Trigger 触发事件，从当前状态转移到目标状态
//   - current: 当前状态（由调用方维护）
//   - event: 触发事件
//   - data: 传递给回调的数据（类型 T）
func (own *FSM[T]) Trigger(current State, event Event, data T) (State, error) {
	atomic.AddUint64(&own.callCount, 1)

	// 1. 获取转移规则
	to, err := own.getDestination(current, event)
	if err != nil {
		return "", err
	}

	// 2. 执行事件回调
	if err := own.runEventCallbacks(event, current, to, data); err != nil {
		if own.stopOnEventError {
			return current, wrapError(ErrEventCallback, "%w", err)
		}
	}

	// 3. 执行离开回调
	if err := own.runLeaveCallbacks(current, data); err != nil {
		if own.stopOnLeaveError {
			return current, wrapError(ErrLeaveCallback, "%w", err)
		}
	}

	// 4. 执行进入回调
	if err := own.runEnterCallbacks(to, data); err != nil {
		if own.stopOnEnterError {
			return current, wrapError(ErrEnterCallback, "%w", err)
		}
	}

	return to, nil
}

// ------------------------------------------------------ 查询相关 ------------------------------------------------------

// GetStates 返回所有状态列表
func (own *FSM[T]) GetStates() []State {
	own.lock.RLock()
	states := make([]State, 0, len(own.states))
	for s := range own.states {
		states = append(states, s)
	}
	own.lock.RUnlock()
	return states
}

// GetTransitions 导出所有转移规则（副本）
func (own *FSM[T]) GetTransitions() map[State]map[Event]State {
	own.lock.RLock()
	result := make(map[State]map[Event]State)
	for from, rules := range own.transitions {
		m := make(map[Event]State, len(rules))
		for _, rule := range rules {
			m[rule.event] = rule.to
		}

		result[from] = m
	}
	own.lock.RUnlock()
	return result
}

// Export 导出完整配置（用于持久化）
func (own *FSM[T]) Export() (states []State, transitions map[State]map[Event]State) {
	states = own.GetStates()
	transitions = own.GetTransitions()
	return
}

// Import 导入配置（会清空现有配置，谨慎使用）
//   - 注意：导入后回调函数会丢失，需要外部重新注册
func (own *FSM[T]) Import(states []State, transitions map[State]map[Event]State) error {
	own.lock.Lock()
	defer own.lock.Unlock()

	newStates := make(map[State]struct{}, len(states))
	for _, s := range states {
		newStates[s] = struct{}{}
	}

	newTrans := make(map[State][]rule, len(transitions))
	for from, m := range transitions {
		rules := make([]rule, 0, len(m))
		for event, to := range m {
			if _, ok := newStates[to]; !ok {
				return wrapStateError(to)
			}

			rules = append(rules, rule{event: event, to: to})
		}

		if _, ok := newStates[from]; !ok {
			return wrapStateError(from)
		}

		newTrans[from] = rules
	}

	own.states = newStates
	own.transitions = newTrans

	// 清空调回调（因为旧回调可能不适用于新状态）
	own.onEvent = make(map[Event][]OnEventHandler[T])
	own.onLeave = make(map[State][]OnLeaveHandler[T])
	own.onEnter = make(map[State][]OnEnterHandler[T])
	return nil
}

// GetStats 返回统计信息
func (own *FSM[T]) GetStats() map[string]any {
	own.lock.RLock()
	states := len(own.states)
	total := 0
	for _, rules := range own.transitions {
		total += len(rules)
	}
	own.lock.RUnlock()

	return map[string]any{
		"states":      states,
		"call_count":  atomic.LoadUint64(&own.callCount),
		"transitions": total,
	}
}

// Validate 验证所有状态和转移的合法性
func (own *FSM[T]) Validate() error {
	own.lock.RLock()
	defer own.lock.RUnlock()

	for from, rules := range own.transitions {
		if _, ok := own.states[from]; !ok {
			return wrapError(ErrTransitionNotFound, "from=%s", from)
		}

		for _, rule := range rules {
			if _, ok := own.states[rule.to]; !ok {
				return wrapError(ErrTransitionNotFound, "from=%s, event=%s, to=%s", from, rule.event, rule.to)
			}
		}
	}

	return nil
}

// ------------------------------------------------------ 内部辅助 ------------------------------------------------------

func (own *FSM[T]) getDestination(from State, event Event) (State, error) {
	own.lock.RLock()
	rules, ok := own.transitions[from]
	if !ok {
		own.lock.RUnlock()
		return "", wrapError(ErrTransitionNotFound, "from=%s", from)
	}

	for _, rule := range rules {
		if rule.event == event {
			own.lock.RUnlock()
			return rule.to, nil
		}
	}

	own.lock.RUnlock()
	return "", wrapError(ErrTransitionNotFound, "from=%s, event=%s", from, event)
}

func (own *FSM[T]) runEventCallbacks(event Event, from, to State, data T) error {
	own.lock.RLock()
	handlers := own.onEvent[event]
	own.lock.RUnlock()

	for _, handle := range handlers {
		if err := handle(from, to, data); err != nil {
			return err
		}
	}

	return nil
}

func (own *FSM[T]) runLeaveCallbacks(state State, data T) error {
	own.lock.RLock()
	handlers := own.onLeave[state]
	own.lock.RUnlock()

	for _, handle := range handlers {
		if err := handle(state, data); err != nil {
			return err
		}
	}

	return nil
}

func (own *FSM[T]) runEnterCallbacks(state State, data T) error {
	own.lock.RLock()
	handlers := own.onEnter[state]
	own.lock.RUnlock()

	for _, handle := range handlers {
		if err := handle(state, data); err != nil {
			return err
		}
	}

	return nil
}
