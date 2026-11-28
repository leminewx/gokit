package tcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"gitee.com/leminewx/gokit/nets"
	"gitee.com/leminewx/gokit/nets/codec"
)

// Client 定义并发安全的 TCP 连接客户端
type Client struct {
	// 基础配置

	host   string      // 设备IP
	addr   string      // 设备地址
	closed atomic.Bool // 连接是否已关闭
	conn   net.Conn    // 网络连接，读写需通过 connMu 保护
	codec  codec.Codec // 数据编解码器

	// 读写超时配置

	dialTimeout  time.Duration // 拨号超时时间
	readTimeout  time.Duration // 读取超时时间
	writeTimeout time.Duration // 写入超时时间

	// 自动重连配置

	maxReconnectRetries    int           // 单次自动重连最大次数：<0 表示无限重连；=0 表示不重连；>0 表示最大重连次数
	reconnectRetryInterval time.Duration // 自动重连失败后，再次重连的间隔时间
	keepAlive              bool          // 是否启用 TCP 保活

	lock      sync.Mutex   // 串行化请求/响应，避免多个 Execute 交错
	connMu    sync.RWMutex // 只保护 conn 指针，使 Close 不必等待进行中的读写
	closeOnce sync.Once
	done      chan struct{} // 关闭信号，用于中断重连/等待过程

	readBytes  atomic.Int64 // 累计读取字节数
	writeBytes atomic.Int64 // 累计写入字节数
	reconnects atomic.Int64 // 成功重连次数
	dialErrors atomic.Int64 // 拨号失败次数
}

// NewClient 新建一个 TCP 客户端
//   - host: 设备IP
//   - port: 设备端口
//   - timeout: 拨号超时时间（可选）
//   - 默认自动重连：关闭
//   - 默认拨号超时时间：10s
//   - 默认读取超时时间：30s
//   - 默认写入超时时间：30s
//   - 默认单次读取最大字节数：4096（可通过配置编解码器修改）
func NewClient(host string, port int, timeout ...time.Duration) (client *Client, err error) {
	var options []Option
	if len(timeout) > 0 && timeout[0] > 0 {
		options = append(options, WithDialTimeout(timeout[0]))
	}

	return NewClientWithOptions(host, port, options...)
}

// NewClientWithOptions 使用选项新建一个 TCP 客户端
func NewClientWithOptions(host string, port int, options ...Option) (client *Client, err error) {
	client = &Client{
		host:         host,
		addr:         fmt.Sprintf("%s:%d", host, port),
		codec:        codec.NewRawCodec(4096),
		dialTimeout:  time.Second * 10,
		readTimeout:  time.Second * 30,
		writeTimeout: time.Second * 30,
		keepAlive:    true,
		done:         make(chan struct{}),
	}

	for _, option := range options {
		option(client)
	}

	conn, err := net.DialTimeout("tcp", client.addr, client.dialTimeout)
	if err != nil {
		return nil, wrapError(ErrDialFailed, "%s: %w", client.addr, err)
	}

	client.setConn(conn)
	return client, nil
}

// SetCodec 设置数据编解码器
func (own *Client) SetCodec(codec codec.Codec) {
	if codec != nil {
		own.lock.Lock()
		own.codec = codec
		own.lock.Unlock()
	}
}

// SetReadTimeout 设置读超时时间
//   - timeout == 0 表示不设置读超时（仅受 ctx 截止时间约束）
//   - 负值被忽略，保留当前配置
func (own *Client) SetReadTimeout(timeout time.Duration) {
	if timeout >= 0 {
		own.lock.Lock()
		own.readTimeout = timeout
		own.lock.Unlock()
	}
}

// SetWriteTimeout 设置写超时时间
//   - timeout == 0 表示不设置写超时（仅受 ctx 截止时间约束）
//   - 负值被忽略，保留当前配置
func (own *Client) SetWriteTimeout(timeout time.Duration) {
	if timeout >= 0 {
		own.lock.Lock()
		own.writeTimeout = timeout
		own.lock.Unlock()
	}
}

// SetReconnect 设置重连参数
//   - interval: 重试间隔（每次重连失败后等待）
//   - retries: 最大重试次数（可选）：省略表示无限重连，0 表示关闭自动重连，>0 表示最大次数
//   - 重连过程会阻塞当前 goroutine，但可通过 Close 立即中断
func (own *Client) SetReconnect(interval time.Duration, retries ...int) {
	own.lock.Lock()
	defer own.lock.Unlock()

	own.applyReconnect(interval, retries...)
}

// applyReconnect 应用重连参数
//   - 未传 retries 表示无限重连；显式传 0 表示关闭自动重连
func (own *Client) applyReconnect(interval time.Duration, retries ...int) {
	if interval >= 0 {
		own.reconnectRetryInterval = interval
	}

	own.maxReconnectRetries = -1
	if len(retries) > 0 {
		if retries[0] > 0 {
			own.maxReconnectRetries = retries[0]
		} else {
			own.maxReconnectRetries = 0
		}
	}
}

// GetAddr 获取连接地址
func (own *Client) GetAddr() string {
	return own.addr
}

// Execute 执行请求，并获取响应
//   - 读取阶段发现连接已断开时，会在新连接上重发完整请求；
//   - 读取超时等非断开错误直接返回，避免重复执行非幂等指令。
func (own *Client) Execute(req []byte, readRespInterval ...time.Duration) (resp []byte, err error) {
	return own.ExecuteContext(context.Background(), req, readRespInterval...)
}

// ExecuteContext 执行请求，并获取响应
//   - 与 Execute 行为一致，额外支持通过 context 取消：等待响应期间取消会立即返回；
//   - 请求串行执行，ctx 不能中断等待其他请求释放串行锁的过程。
func (own *Client) ExecuteContext(ctx context.Context, req []byte, readRespInterval ...time.Duration) (resp []byte, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	own.lock.Lock()
	defer own.lock.Unlock()

	return own.execute(ctx, req, readRespInterval...)
}

// execute 在持有串行锁的前提下执行一次请求
func (own *Client) execute(ctx context.Context, req []byte, readRespInterval ...time.Duration) (resp []byte, err error) {
	if own.closed.Load() {
		return nil, ErrConnClosed
	}

	retries := 0
	for {
		if err = ctx.Err(); err != nil {
			return nil, err
		}

		if err = own.write(ctx, req); err != nil {
			return nil, err
		}

		if len(readRespInterval) > 0 && readRespInterval[0] > 0 {
			if !own.sleep(ctx, readRespInterval[0]) {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return nil, ctxErr
				}
				return nil, wrapError(ErrConnClosed, "while waiting response")
			}
		}

		resp, err = own.readOnce(ctx)
		if err == nil {
			return resp, nil
		}

		// ctx 取消/超时优先返回上下文错误，避免误导为连接异常
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		// 仅连接断开才重发；readOnce 已将旧连接标记为失效，下一轮 write 会先重连
		if !isConnBroken(err) || own.maxReconnectRetries == 0 {
			return nil, err
		}

		if own.maxReconnectRetries > 0 && retries >= own.maxReconnectRetries {
			return nil, wrapError(ErrReconnectFailed, "after %d attempts: %w", retries, err)
		}

		retries++
	}
}

// Write 写入数据
func (own *Client) Write(data []byte) error {
	own.lock.Lock()
	defer own.lock.Unlock()
	return own.write(context.Background(), data)
}

func (own *Client) write(ctx context.Context, data []byte) (err error) {
	// 检查是否已经关闭
	if own.closed.Load() {
		return ErrConnClosed
	}
	if err = ctx.Err(); err != nil {
		return err
	}

	// 重试循环（支持自动重连）
	for {
		conn := own.getConn()

		// 连接已被标记失效时先重连
		if conn == nil {
			if own.maxReconnectRetries == 0 {
				return ErrReconnectDisabled
			}

			if err = own.dial(ctx); err != nil {
				return err
			}

			if conn = own.getConn(); conn == nil {
				return ErrNotConnected
			}
		}

		// 设置写超时（取超时与 ctx 截止时间的较早者）
		if err = conn.SetWriteDeadline(own.deadline(ctx, own.writeTimeout)); err != nil {
			own.invalidate(conn)
			return wrapError(ErrSetDeadline, "write: %w", err)
		}

		// 尝试写入
		if err = own.codec.Encode(conn, data); err == nil {
			own.writeBytes.Add(int64(len(data)))
			return nil // 成功
		}

		// 写入失败说明当前连接不可用，丢弃后按连接错误分类
		own.invalidate(conn)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if !nets.IsConnectionError(err) || own.maxReconnectRetries == 0 {
			return err
		}
	}
}

// Read 读取数据
func (own *Client) Read() ([]byte, error) {
	own.lock.Lock()
	defer own.lock.Unlock()
	return own.readOnce(context.Background())
}

// readOnce 读取一次数据；失败时标记连接失效，防止后续请求读到上一次的迟到响应
func (own *Client) readOnce(ctx context.Context) (data []byte, err error) {
	// 检查是否已经关闭
	if own.closed.Load() {
		return nil, ErrConnClosed
	}

	conn := own.getConn()
	if conn == nil {
		return nil, ErrNotConnected
	}

	// 设置读超时时间（取超时与 ctx 截止时间的较早者）
	if err = conn.SetReadDeadline(own.deadline(ctx, own.readTimeout)); err != nil {
		own.invalidate(conn)
		return nil, wrapError(ErrSetDeadline, "read: %w", err)
	}

	// 读取数据
	data, err = own.codec.Decode(conn)
	if err != nil {
		own.invalidate(conn)
		return nil, err
	}

	own.readBytes.Add(int64(len(data)))
	return data, nil
}

// deadline 计算读写截止时间：取超时与 ctx 截止时间的较早者
//   - timeout <= 0 表示不设置超时；若无 ctx 截止时间则返回零值（即无 deadline）
//   - 若仅是直接 time.Now().Add(0)，会得到"已经过期"的截止时间，使读/写立即超时
func (own *Client) deadline(ctx context.Context, timeout time.Duration) time.Time {
	contextDeadline, hasContextDeadline := ctx.Deadline()
	if timeout <= 0 {
		if hasContextDeadline {
			return contextDeadline
		}

		return time.Time{}
	}

	deadline := time.Now().Add(timeout)
	if hasContextDeadline && contextDeadline.Before(deadline) {
		return contextDeadline
	}

	return deadline
}

// dial 建立新连接（调用前必须持有锁）
func (own *Client) dial(ctx context.Context) error {
	maxRetries := own.maxReconnectRetries
	if maxRetries < 0 {
		maxRetries = math.MaxInt
	}
	if maxRetries == 0 {
		maxRetries = 1
	}

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// 首次重试不等待，后续每次失败后等待
		if attempt > 0 && own.reconnectRetryInterval > 0 {
			if !own.sleep(ctx, own.reconnectRetryInterval) {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return ErrReconnectInterrupted
			}
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		// 重连过程中如果连接被用户关闭，立即返回
		if own.closed.Load() {
			return wrapError(ErrConnClosed, "cannot reconnect")
		}

		// 拨号超时，避免无限阻塞
		conn, dialErr := net.DialTimeout("tcp", own.addr, own.dialTimeout)
		if dialErr != nil {
			own.dialErrors.Add(1)
			err = dialErr
			continue
		}

		// 拨号期间客户端可能已被关闭，避免泄漏连接
		if own.closed.Load() {
			_ = conn.Close()
			return ErrReconnectInterrupted
		}

		own.reconnects.Add(1)
		own.setConn(conn)
		return nil // 重连成功
	}

	return wrapError(ErrReconnectFailed, "after %d attempts: %w", maxRetries, err)
}

// getConn 获取当前连接
func (own *Client) getConn() net.Conn {
	own.connMu.RLock()
	defer own.connMu.RUnlock()
	return own.conn
}

// setConn 保存新连接
func (own *Client) setConn(conn net.Conn) {
	own.applySocketOptions(conn)

	own.connMu.Lock()
	own.conn = conn
	own.connMu.Unlock()
}

// applySocketOptions 应用保活等套接字选项
func (own *Client) applySocketOptions(conn net.Conn) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}

	_ = tcpConn.SetKeepAlive(own.keepAlive)
	_ = tcpConn.SetNoDelay(true)
}

// takeConn 取出并清空当前连接
func (own *Client) takeConn() net.Conn {
	own.connMu.Lock()
	conn := own.conn
	own.conn = nil
	own.connMu.Unlock()
	return conn
}

// invalidate 丢弃指定连接；仅当它仍是当前连接时才清空，避免误伤刚重连的新连接
func (own *Client) invalidate(conn net.Conn) {
	if conn == nil {
		return
	}

	_ = conn.Close()

	own.connMu.Lock()
	if own.conn == conn {
		own.conn = nil
	}
	own.connMu.Unlock()
}

// sleep 等待指定时长，期间若客户端关闭或 ctx 取消则返回 false（调用前必须持有锁）
func (own *Client) sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-own.done:
		return false
	case <-ctx.Done():
		return false
	}
}

// isConnBroken 判断读取失败是否属于连接断开
//   - 超时不视为断开：此时响应可能只是迟到，重发会导致非幂等指令被重复执行
func isConnBroken(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE)
}

// Close 关闭连接
//   - 可重复调用；不再等待进行中的读写，会立即关闭底层连接使其返回
func (own *Client) Close() error {
	own.closeOnce.Do(func() { close(own.done) })
	own.closed.Store(true)

	conn := own.takeConn()
	if conn == nil {
		return nil
	}

	if err := conn.Close(); err != nil {
		return wrapError(ErrCloseFailed, "%w", err)
	}

	return nil
}
