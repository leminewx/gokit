package mesh

import (
	"log"
	"time"
)

// ******************************************************************************************************************
// Config：全局配置模型
// ******************************************************************************************************************

type (
	Config struct {
		// 调试模式
		Debug bool

		// 消息多跳广播的生存时间
		//   - 每经过一次转发减 1，当 TTL <= 0 时不再转发
		MessageTTL int
		// 消息发送通道缓冲大小
		MessageSendChanSize int
		// 发送队列满时单播的等待时长
		//   - 0 表示不等待，直接返回 ErrSendQueueFull（背压：丢弃但不阻塞调用方）
		//   - > 0 表示最多等待该时长，用于对可靠性要求高于延迟的场景
		SendBlockTimeout time.Duration
		// 消息编解码器，缺省 NewJSONCodec()
		Codec Codec

		// 处理消息序列号的最大保留时间（用于定期清理 seen map）
		MaxSeenAge time.Duration
		// 请求超时时间（用于 Get/Query）
		RequestTimeout time.Duration
		// 连接拨号超时时间
		ConnDialTimeout time.Duration

		// Gossip 广播间隔
		GossipInterval time.Duration
		// Gossip 节点广播数量（每次随机选择几个节点发送心跳）
		GossipFanoutSize int
		// Gossip 节点部分视图数量（每次心跳携带多少个节点信息）
		GossipPartialViewSize int
		// Gossip 节点故障超时
		GossipFailureTimeout time.Duration

		// 种子节点
		seedAddresses []string
	}

	Option func(*Config)
)

const (
	_PEER_SCORE_INTERVAL    time.Duration = 1 * time.Hour    // 分数衰减间隔
	_PEER_PROTECTION_PERIOD time.Duration = 30 * time.Second // 新连接保护期

	// _PEER_MAINTENANCE_SCORE_INTERVAL 维护加分补偿的间隔
	_PEER_MAINTENANCE_SCORE_INTERVAL time.Duration = 10 * time.Minute

	_PEER_SCORE_MAX                       int64   = 10000 // 最大分数
	_PEER_SCORE_BONUS                     int64   = 1000  // 种子节点激励分数
	_PEER_ACTIVE_DELTA                    int64   = 1     // 每次收到消息增加的分数（防刷限制）
	_PEER_SCORE_INIT_BASE                 int64   = 100   // 连接基础分数
	_PEER_MAX_ACTIVE_INCREMENT_PER_MINUTE int64   = 10    // 每分钟最多增加10次活跃分
	_PEER_SOFT_LIMIT_FACTOR               float64 = 1.2   // 软限制因子，允许超出 maxPeers 20%
	_PEER_SCORE_DECAY_FACTOR              float64 = 0.9   // 分数衰减因子
)

// NewDefaultConfig 创建默认配置
//   - Debug = false：关闭调试模式

func NewDefaultConfig() *Config {
	return &Config{
		Debug:                 false,
		MaxSeenAge:            1 * time.Minute,
		MessageSendChanSize:   128,
		Codec:                 NewJSONCodec(),
		RequestTimeout:        5 * time.Second,
		ConnDialTimeout:       5 * time.Second,
		GossipInterval:        2 * time.Second,
		GossipFanoutSize:      3,
		GossipPartialViewSize: 20,
		GossipFailureTimeout:  15 * time.Second,
	}
}

// WithDebug 启用调试模式
//   - 默认关闭
func WithDebug() Option {
	return func(config *Config) {
		config.Debug = true
	}
}

// WithMessageTTL 设置消息的 TTL，每经过一次转发减 1（用于多跳广播）
//   - 当 TTL <= 0 时不再转发
//   - 用于限制广播范围，防止消息无限扩散，仅对需要多跳广播的消息有效
func WithMessageTTL(ttl int) Option {
	return func(config *Config) {
		if ttl > 0 {
			config.MessageTTL = ttl
		}
	}
}

// WithMessageSendChanSize 设置消息发送通道的缓冲大小
//   - 默认 128
func WithMessageSendChanSize(size int) Option {
	return func(config *Config) {
		if size > 0 {
			config.MessageSendChanSize = size
		}
	}
}

// WithMaxSeenAge 设置消息序列号的最大保留时间（用于定期清理 seen map）
//   - 默认 1 分钟
func WithMaxSeenAge(age time.Duration) Option {
	return func(config *Config) {
		if age > 0 {
			config.MaxSeenAge = age
		}
	}
}

// WithCodec 设置消息编解码器
//   - 缺省使用 NewJSONCodec()，编解码器必须自带消息边界且无状态
func WithCodec(codec Codec) Option {
	return func(config *Config) {
		if codec != nil {
			config.Codec = codec
		}
	}
}

// WithSendBlockTimeout 设置发送队列满时单播的等待时长
//   - 0（缺省）表示不等待，立即返回 ErrSendQueueFull
//   - > 0 表示最多等待该时长，超时后仍返回 ErrSendQueueFull
//   - 广播始终不等待，仅跳过已满的连接，避免慢节点拖垮整个网络
func WithSendBlockTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		if timeout > 0 {
			config.SendBlockTimeout = timeout
		}
	}
}

// codec 返回实际生效的编解码器，未配置时回退到 JSON
func (own *Config) codec() Codec {
	if own.Codec == nil {
		return NewJSONCodec()
	}

	return own.Codec
}

// WithRequestTimeout 设置请求超时时间（用于 Get/Query）
//   - 默认 5 秒
func WithRequestTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		if timeout > 0 {
			config.RequestTimeout = timeout
		}
	}
}

// WithConnDialTimeout 设置连接拨号超时时间
//   - 默认 5 秒
func WithConnDialTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		if timeout > 0 {
			config.ConnDialTimeout = timeout
		}
	}
}

// WithGossipInterval 设置 Gossip 任务间隔时间
//   - 默认 2 秒
func WithGossipInterval(interval time.Duration) Option {
	return func(config *Config) {
		if interval > 0 {
			config.GossipInterval = interval
		}
	}
}

// WithGossipFanoutSize 设置 Gossip 扇出数量（每次随机选择几个节点发送心跳）
//
// 扇出机制：
//   - 用最小的网络开销，让信息在尽可能短的时间内扩散到整个集群
//   - 每个节点定期随机选择少量节点传播信息，收到信息的节点再继续传播，类似于病毒传播
//
// 扇出数量：3（默认值）
//   - 1 个节点：传播链式依赖，如果某个节点失败，信息可能无法传播到整个网络，且需要 O(N) 轮，延迟线性增长
//   - 2 个节点：经典值，比如 Bitcoin 的 gossip 使用 2-3 个节点（传播轮数约 log₂ N，对于 1000 节点约 10 轮）
//   - 3 个节点：传播轮数约 log₃1000 ≈ 6.3 轮，比 2 个节点快约 37%，而每轮消息量仅增加 50%（延迟和开销的黄金平衡点）
//   - 4 个节点：收益递减 log₄1000 ≈ 5 轮，仅比 3 个节点少 1 轮，但消息量增加 33%。
//   - 5+ 个节点：接近全连接，消息总量 O(N²)，失去了 gossip 的可扩展性
func WithGossipFanoutSize(num int) Option {
	return func(config *Config) {
		if num > 0 {
			config.GossipFanoutSize = num
		}
	}
}

// WithGossipPartialViewSize 设置 Gossip 部分视图数量（每次心跳随机携带几个节点信息）
//
// 部分视图机制：
//   - 从本地已知的所有节点中，抽取一个固定大小的子集（样本），附加在心跳消息中发送给其他节点
//   - 接收方利用这个样本来更新自己的成员表，从而实现全网成员信息的最终一致性
//
// 部分视图数量：20（默认值）
//   - 控制单条消息的大小
//   - 平衡收敛速度与开销（每个周期发送 O(logN) 个节点的信息，就能以高概率在 O(logN) 轮内让全网节点收敛）
//   - 避免消息过大导致分片或丢包
//   - 工程惯例（Cassandra、Serf、Memberlist 等默认使用 20~30 的样本大小）
func WithGossipPartialViewSize(num int) Option {
	return func(config *Config) {
		if num > 0 {
			config.GossipPartialViewSize = num
		}
	}
}

// WithGossipFailureTimeout 配置 Gossip 失败超时时间
func WithGossipFailureTimeout(timeout time.Duration) Option {
	return func(config *Config) {
		if timeout > 0 {
			config.GossipFailureTimeout = timeout
		}
	}
}

const _LOG_PREFIX = "[DEBUG] [mesh] "

func (own *Config) logDebug(format string, args ...any) {
	if own.Debug {
		if len(args) == 0 {
			log.Println(_LOG_PREFIX + format)
		} else {
			log.Printf(_LOG_PREFIX+format, args...)
		}
	}
}
