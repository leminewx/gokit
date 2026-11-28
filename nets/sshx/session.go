package sshx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"gitee.com/leminewx/gokit/stringx"
	"golang.org/x/crypto/ssh"
)

// Command 非流式执行命令
type Command struct {
	Command        string                  // 命令
	Timeout        time.Duration           // 执行超时时间
	Terminal       Terminal                // 伪终端终端配置
	InputArgs      []string                // 输入参数
	InputInterval  time.Duration           // 参数输入间隔时间
	QuitSignals    []string                // 自定义退出信号（默认使用 ["\x03", "\x1C"]）
	QuitTimeout    time.Duration           // 退出信号超时时间（默认 500 毫秒）
	StderrFilter   func(stderr error) bool // 忽略指定错误通道返回的错误（true 表示忽略错误）
	ExitCodeFilter func(code int) bool     // 忽略指定退出码错误（true 表示忽略错误）

	stdin     io.WriteCloser
	doneChan  <-chan error
	closeChan chan<- struct{}
}

// Execute 执行 SSH 命令，并返回执行结果
func (own *Client) Execute(cmd *Command) ([]byte, error) {
	return own.ExecuteContext(context.Background(), cmd)
}

// ExecuteContext 执行上下文命令，并返回执行结果
func (own *Client) ExecuteContext(ctx context.Context, cmd *Command) ([]byte, error) {
	// 参数校验
	if cmd == nil || cmd.Command == "" {
		return nil, ErrEmptyCommand
	}
	if len(cmd.QuitSignals) == 0 {
		cmd.QuitSignals = []string{"\x03", "\x1C"}
	}
	if cmd.QuitTimeout <= 0 {
		cmd.QuitTimeout = 500 * time.Millisecond
	}

	if own.IsClosed() {
		return nil, ErrClosed
	}

	// 创建 SSH 会话
	sess, err := own.newSession()
	if err != nil {
		return nil, err
	}

	start := time.Now()

	// ctx 为 nil 时使用 Background；与 Command.Timeout 同时生效时取较早的截止时间
	if ctx == nil {
		ctx = context.Background()
	}
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	stdout, stderr, err := own.executeSession(ctx, sess, cmd)
	return own.handleOutput(cmd, time.Since(start), stdout, stderr, err)
}

// executeSession 通过 SSH 会话执行命令，并返回执行结果（返回原生的错误消息，不封装）
func (own *Client) executeSession(ctx context.Context, session *ssh.Session, cmd *Command) (stdout, stderr []byte, err error) {
	// 1. 设置 stdout/stderr 缓冲区
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// 2. 创建 stdin 管道（用于输入和发送 Ctrl+C）
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		atomic.AddInt64(&own.sessionTotal, -1)
		return nil, nil, wrapError(ErrSession, "create stdin pipe: %w", err)
	}
	cmd.stdin = stdin

	// 3. 初始化 PTY 伪终端
	if err := cmd.Terminal.Init(session); err != nil {
		_ = session.Close()
		atomic.AddInt64(&own.sessionTotal, -1)
		return nil, nil, wrapError(ErrSession, "init pty terminal: %w", err)
	}
	own._debug("Init PTY terminal successfully")

	// 4. 启动会话命令
	if err := session.Start(cmd.Command); err != nil {
		_ = session.Close()
		atomic.AddInt64(&own.sessionTotal, -1)
		return nil, nil, wrapError(ErrSession, "start session: %w", err)
	}
	own._debug("Start to execute session >>> %s", cmd.Command)

	// 5. 异步处理输入
	wg := &sync.WaitGroup{}
	if len(cmd.InputArgs) > 0 {
		wg.Go(func() {
			// 1. 发送命令参数
			if err := own.inputArguments(ctx, cmd, nil); err != nil {
				own._debug(err.Error())
			}

			// 2. 发送 EOF (Ctrl+D) 通知输入结束，但不关闭管道
			if _, err := cmd.stdin.Write([]byte("\x04")); err != nil {
				own._debug("Failed to input EOF (Ctrl+D): %v", err)
			} else {
				own._debug("Input EOF (Ctrl+D)")
			}
		})
	}

	// 6. 异步等待会话结束
	done := make(chan error, 1)
	closed := make(chan struct{}, 1)
	wg.Go(func() {
		defer atomic.AddInt64(&own.sessionTotal, -1)
		own.waitSession(session, closed, done)
		own._debug("Session done")
	})

	// 7. 阻塞等待执行结束
	select {
	case err = <-done: // 执行正常结束
	case <-ctx.Done(): // 上下文取消执行
		cmd.doneChan = done
		cmd.closeChan = closed
		err = own.shutdownSession(ctx, session, cmd)
	}

	// 8. 等待所有 goroutine 结束，防止溢出
	wg.Wait()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

// inputArguments 输入参数，内部会在退出前关闭 stdin
func (own *Client) inputArguments(ctx context.Context, cmd *Command, handle func(input string) string) error {
	for _, input := range cmd.InputArgs {
		select {
		case <-ctx.Done():
			return nil
		default:
			// 1. 输入间隔或上下文结束
			if cmd.InputInterval > 0 {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(cmd.InputInterval):
				}
			}

			// 2. 处理输入参数
			if handle != nil {
				input = handle(input)
			}

			// 3. 输入参数
			if _, err := fmt.Fprintln(cmd.stdin, input); err != nil {
				return wrapError(ErrWrite, "%w", err)
			}
			// 输入内容可能是口令等敏感信息，调试日志只记录长度
			own._debug("Input arguments >>> [REDACTED, %d bytes]", len(input))
		}
	}

	return nil
}

// waitSession 异步等待会话结束，并把执行结果写入 done 通道
//   - 交由 execute 或 shutdownSession 统一读取与处理，避免阻塞调用 goroutine
func (own *Client) waitSession(sess *ssh.Session, closed <-chan struct{}, done chan<- error) {
	defer func() {
		if r := recover(); r != nil {
			// 记录 panic，并确保 done 被写入（例如写入一个错误）
			done <- wrapError(ErrPanic, "%v", r)
		}
	}()

	// 1. 异步执行，确保当前主 goroutine 一定可以退出
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- sess.Wait()
	}()

	// 2. 执行完成或主动关闭客户端
	select {
	case err := <-waitDone:
		done <- err
	case <-closed:
		// 此时 session.Wait() 已经阻塞，如不处理，则可能溢出
		// 若当前客户端只有一个会话，则通过关闭客户端来关闭当前会话，后续调用通过重新连接在正常使用
		// 已知限制：只有 sessionTotal == 1 时才会关闭客户端，存在并发会话时等待 goroutine
		// 需等到其它会话结束才能退出，调用方应避免在同一客户端上长期挂着多个会话
		own.lock.Lock()
		if own.client != nil && atomic.LoadInt64(&own.sessionTotal) == 1 {
			_ = own.client.Close()
			own._debug("Closing client due to session blocking, and subsequent usage will automatically reconnect and restore")
		}
		own.lock.Unlock()
	}
}

// shutdownSession 逐步发送信号关闭会话，返回最终错误
func (own *Client) shutdownSession(ctx context.Context, session *ssh.Session, cmd *Command) error {
	// 1. 发送信号关闭信号
	if cmd.stdin != nil {
		for _, sig := range cmd.QuitSignals {
			if _, err := cmd.stdin.Write([]byte(sig)); err == nil {
				break
			}
		}

		own._debug("Closed stdin pipe")
		_ = cmd.stdin.Close()
	}

	// 2. 等待会话关闭退出或超时
	select {
	case err := <-cmd.doneChan: // 会话终止结束
		own._debug("Session terminated")
		return err
	case <-time.After(300 * time.Millisecond): // 会话终止超时，执行后续终止操作
		own._debug("Session termination timeout: 300ms")
		_ = session.Signal(ssh.SIGINT)
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Signal(ssh.SIGTERM)
	}
	_ = session.Close()

	// 3. 等待会话最终结束
	select {
	case err := <-cmd.doneChan:
		own._debug("Session closed")
		return err
	case <-time.After(max(cmd.QuitTimeout-300*time.Millisecond, 100*time.Millisecond)):
		cmd.closeChan <- struct{}{}
		own._debug("Session close timeout: %s", cmd.QuitTimeout-300*time.Millisecond)
		return ctx.Err()
	}
}

// handleOutput 统一输出处理
func (own *Client) handleOutput(cmd *Command, cost time.Duration, stdout, stderr []byte, err error) ([]byte, error) {
	// 删除 ANSI 转义符
	stdout = stringx.StripANSIBytes(stdout)
	own._debug("Receive stdout >>> [%v]\n%s\n", cost, stdout)

	stderr = stringx.StripANSIBytes(stderr)
	if len(stderr) > 0 {
		own._debug("Receive stderr >>> [%v]\n%s\n", cost, stderr)
	}

	// 处理命令执行错误（包括退出码非零）
	if err != nil {
		// 检查是否为退出错误码，并应用过滤器
		var exitErr *ssh.ExitError
		if errors.As(err, &exitErr) && cmd.ExitCodeFilter != nil && cmd.ExitCodeFilter(exitErr.ExitStatus()) {
			// 过滤器认为该退出码不应作为错误，返回 stdout 和 nil
			return stdout, nil
		}

		// 附加 stderr 到错误信息（如果存在）
		if len(stderr) > 0 {
			return stdout, wrapError(ErrCommandFailed, "%w; stderr: %s", err, stderr)
		}

		return stdout, err
	}

	// 命令成功执行（退出码为 0）
	if len(stderr) > 0 {
		err = wrapError(ErrUnexpectedOutput, "%s", stderr)

		// 根据 StderrFilter 决定是否将 stderr 视为错误
		if cmd.StderrFilter != nil && cmd.StderrFilter(err) {
			return stdout, nil
		}

		// 视为错误：返回 stderr 和构造的错误
		return stdout, err
	}

	// 成功，返回 stdout
	return stdout, nil
}
