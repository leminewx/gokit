package sshx

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/leminewx/gokit/nets"
	"golang.org/x/crypto/ssh"
)

// Client 定义 SSH 客户端的数据结构
type Client struct {
	// 基础配置

	host      string        // 主机 IP
	addr      string        // 地址 host:port
	debug     *atomic.Bool  // 调试模式
	debugSize *atomic.Int64 // 调试输出最大长度
	closed    *atomic.Bool  // 连接状态标记
	optionErr error         // 选项解析错误（如 known_hosts 加载失败）

	// 客户端配置

	config *ssh.ClientConfig // SSH 配置
	client *ssh.Client       // SSH 客户端

	// 自动重连配置

	sessionTotal           int64         // 当前会话数
	maxReconnectRetries    int           // 单次自动重连最大次数：<0 表示无限重连；=0 表示不重连；>0 表示最大重连次数
	reconnectRetryInterval time.Duration // 自动重连失败后，再次重连的间隔时间

	lock sync.RWMutex
}

// NewClient 新建一个 SSH 客户端，并指定账号和密码
//   - host: 主机 IP
//   - user: 用户名
//   - pwd: 密码
//   - 默认端口为 22
//   - 默认拨号超时 30 秒
//   - 默认开启重连（重连间隔 5 秒，最大连续重连次数 10 次）
func NewClient(host, user, pwd string, opts ...Option) (*Client, error) {
	client := &Client{
		host:      host,
		addr:      fmt.Sprintf("%s:%d", host, 22),
		debug:     &atomic.Bool{},
		debugSize: &atomic.Int64{},
		closed:    &atomic.Bool{},
		config: &ssh.ClientConfig{
			User:            user,
			Auth:            []ssh.AuthMethod{ssh.Password(pwd)},
			Timeout:         time.Second * 30,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
		maxReconnectRetries:    10,
		reconnectRetryInterval: time.Second * 5,
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.optionErr != nil {
		return nil, client.optionErr
	}

	var err error
	if client.client, err = ssh.Dial("tcp", client.addr, client.config); err != nil {
		return nil, wrapError(ErrConnect, "%w", err)
	}

	client._debug("Connected to SSH server successfully")
	return client, nil
}

// DisableDebug 关闭调试模式
func (own *Client) DisableDebug() {
	own.debug.Store(false)
	own.debugSize.Store(0)
}

// SetReconnect 设置重连参数
//   - retries: 最大连续重试次数（-1 表示无限重连；0 表示不启用重连；>0 表示最大重连次数，默认值 10）
//   - interval: 重试间隔（每次重连失败后等待，支持 0）
//   - 重连过程会阻塞当前 goroutine 并持有内部锁，在此期间其他所有操作会被阻塞，直到重连成功或失败；其他所有会话会被重置
func (own *Client) SetReconnect(retries int, interval ...time.Duration) {
	own.lock.Lock()
	own.maxReconnectRetries = retries
	if len(interval) > 0 && interval[0] >= 0 {
		own.reconnectRetryInterval = interval[0]
	}
	own.lock.Unlock()
}

// GetClient 获取 SSH 客户端
//   - 返回的句柄在重连或关闭后可能失效，调用方需自行处理相关错误
func (own *Client) GetClient() *ssh.Client {
	own.lock.RLock()
	defer own.lock.RUnlock()

	return own.client
}

// IsClosed 检查连接是否已关闭
func (own *Client) IsClosed() bool {
	return own.closed.Load()
}

// newSession 创建 SSH 会话
func (own *Client) newSession() (*ssh.Session, error) {
	// 检查是否已经关闭
	if own.IsClosed() {
		return nil, ErrClosed
	}

	own.lock.Lock()
	defer own.lock.Unlock()

	// 重试循环（支持自动重连）
	//   - 最多重连一次：重连成功后仍然失败则直接返回，避免无限自旋
	for attempt := 0; ; attempt++ {
		if own.client == nil {
			if attempt > 0 {
				return nil, ErrNoConnection
			}

			if err := own.reconnect(nil); err != nil {
				return nil, err
			}

			continue
		}

		// 设置读超时时间
		sess, err := own.client.NewSession()
		if err == nil {
			atomic.AddInt64(&own.sessionTotal, 1)
			own._debug("Created new session")
			return sess, nil
		}

		if attempt > 0 {
			return nil, err
		}

		// 尝试重连
		if err := own.reconnect(err); err != nil {
			return nil, err
		}
	}
}

// RunCommand 执行命令并输出结果
//   - 注意：只能执行非阻塞性命令，若要执行阻塞型命令，推荐使用 Execute 或 ExecuteStream 方法
func (own *Client) RunCommand(cmd string) ([]byte, error) {
	sess, err := own.newSession()
	if err != nil {
		return nil, err
	}

	defer func() {
		atomic.AddInt64(&own.sessionTotal, -1)
		_ = sess.Close()
	}()

	own._debug("Start to execute command >>> %s", cmd)
	start := time.Now()
	res, err := sess.Output(cmd)
	own._debug("Output done")

	return own.handleOutput(&Command{Command: cmd}, time.Since(start), res, []byte(""), err)
}

// reconnect 尝试重新建立连接，调用前必须持有锁（重连前已关闭旧客户端，确保无活动会话）
func (own *Client) reconnect(err error) error {
	// 已经关闭，不能重新连接
	if own.closed.Load() {
		return wrapError(ErrReconnect, "%w", ErrClosed)
	}

	// 非连接错误，返回错误；nil 表示当前没有可用连接，直接重连
	if err != nil && !nets.IsConnectionError(err) {
		return err
	}

	// 不支持重连，返回错误
	if own.maxReconnectRetries == 0 {
		return err
	}

	// 关闭客户端
	if own.client != nil {
		_ = own.client.Close()
	}

	// 初始化最大重连次数
	own._debug("Try to reconnect...")
	maxRetries := own.maxReconnectRetries
	if own.maxReconnectRetries < 0 {
		maxRetries = int(^uint(0) >> 1) // 实际上相当于无限，但避免整数溢出
	}

	for retries := 0; retries < maxRetries; retries++ {
		// Close 可能已置关闭标记，及时退出避免长时间持锁
		if own.closed.Load() {
			return wrapError(ErrReconnect, "%w", ErrClosed)
		}

		// 如果是第一次重试不等待；否则等待间隔
		if retries > 0 && own.reconnectRetryInterval > 0 {
			if !own.waitReconnectInterval(own.reconnectRetryInterval) {
				return wrapError(ErrReconnect, "%w", ErrClosed)
			}
		}

		// 拨号超时，避免无限阻塞
		client, dialErr := ssh.Dial("tcp", own.addr, own.config)
		if dialErr == nil {
			own.client = client
			atomic.StoreInt64(&own.sessionTotal, 0)
			own._debug("SSH client is reconnected, retries: %d", retries+1)
			return nil // 重连成功
		}

		err = dialErr
		own._debug("Failed to reconnect, retries: %d, error: %v", retries+1, err)
	}

	if err == nil {
		return wrapError(ErrReconnect, "after %d attempts", maxRetries)
	}

	return wrapError(ErrReconnect, "after %d attempts: %w", maxRetries, err)
}

// waitReconnectInterval 等待重连间隔，期间定期检查关闭状态，使 Close 能尽快生效
//   - 返回 false 表示客户端已关闭，应中断重连
func (own *Client) waitReconnectInterval(interval time.Duration) bool {
	const step = 100 * time.Millisecond

	deadline := time.Now().Add(interval)
	for {
		if own.closed.Load() {
			return false
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return true
		}
		if remaining > step {
			remaining = step
		}
		time.Sleep(remaining)
	}
}

// Close 关闭 SSH 客户端
//   - 可重复调用；重连过程中调用可使其尽快退出
func (own *Client) Close() error {
	// 先置关闭标记：重连会持有锁，不能先等锁，否则 Close 会被长时间阻塞
	if own.closed.Swap(true) {
		return nil
	}

	own.lock.Lock()
	defer own.lock.Unlock()

	if own.client == nil {
		own._debug("SSH client is closed")
		return nil
	}

	// 关闭客户端
	if err := own.client.Close(); err != nil {
		return wrapError(ErrClose, "%w", err)
	}

	own._debug("SSH client is closed")
	return nil
}

func (own *Client) _debug(format string, args ...any) {
	if !own.debug.Load() {
		return
	}

	info := fmt.Sprintf("[DEBUG] [SSH] [%s] "+format, append([]any{own.addr}, args...)...)
	size := own.debugSize.Load()
	// 统一用 int64 比较，避免 32 位平台上的 int 溢出；同时排除非正数，防止切片越界
	if size <= 0 || size >= int64(len(info)) {
		log.Println(info)
		return
	}

	log.Println(info[:size] + "......")
}
