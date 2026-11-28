package sshx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// StreamCommand 流式执行命令
type StreamCommand struct {
	Command                     string        // 命令
	Timeout                     time.Duration // 执行超时时间
	Terminal                    Terminal      // 终端参数
	InputArgs                   []string      // 输入参数
	InputInterval               time.Duration // 参数输入间隔时间
	QuitSignals                 []string      // 自定义退出信号（默认使用 ["\x03", "\x1C"]）
	QuitTimeout                 time.Duration // 信号超时时间（默认 500 毫秒）
	StdoutReadDelimit           byte          // 标准输出分隔符（默认换行符）
	StdoutReadTimeoutAfterClose time.Duration // 会话结束后，若 stdout 缓存数据未读取完，则可以超时读取（默认 1 秒）
}

// StreamHandler 定义输出流处理器的接口类型（如需并发调用，则需实现并发安全）
type StreamHandler interface {
	// CallOnStdin 处理输入时调用
	CallOnStdin(input string) string
	// CallOnStdout 处理标准输出时调用（支持自定义分隔方式）
	CallOnStdout(line string)
	// CallOnStderr 处理错误输出时调用（使用换行符分隔）
	CallOnStderr(line string)
	// CallOnDone 处理执行完成时调用（session.Wait() 的返回）
	CallOnDone(err error)
	// CallOnError 处理执行过程中的错误时调用
	CallOnError(err error)
}

// ExecuteStream 流式执行命令（通过流处理器处理输入和输出）
func (own *Client) ExecuteStream(cmd *StreamCommand, handler StreamHandler) {
	if cmd.Timeout == 0 {
		own.ExecuteStreamContext(context.Background(), cmd, handler)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), cmd.Timeout)
	defer cancel()

	own.ExecuteStreamContext(ctx, cmd, handler)
}

// ExecuteStreamContext 流式执行带上下的文命令（通过流处理器处理输入和输出）
func (own *Client) ExecuteStreamContext(ctx context.Context, cmd *StreamCommand, handler StreamHandler) {
	// 参数校验
	if cmd == nil || cmd.Command == "" {
		handler.CallOnError(ErrEmptyCommand)
		return
	}
	if len(cmd.QuitSignals) == 0 {
		cmd.QuitSignals = []string{"\x03", "\x1C"}
	}
	if cmd.QuitTimeout <= 0 {
		cmd.QuitTimeout = 500 * time.Millisecond
	}
	if cmd.StdoutReadDelimit == 0 {
		cmd.StdoutReadDelimit = '\n'
	}
	if cmd.StdoutReadTimeoutAfterClose <= 0 {
		cmd.StdoutReadTimeoutAfterClose = time.Second
	}

	if own.IsClosed() {
		handler.CallOnError(ErrClosed)
		return
	}

	// 创建 SSH 会话
	sess, err := own.newSession()
	if err != nil {
		handler.CallOnError(err)
		return
	}

	// 携带上下文
	if ctx != nil {
		own.executeStream(ctx, sess, cmd, handler)
		return
	}

	// 未设置超时时间
	ctx = context.Background()
	if cmd.Timeout <= 0 {
		own.executeStream(ctx, sess, cmd, handler)
		return
	}

	// 携带超时控制
	ctx, cancel := context.WithTimeout(ctx, cmd.Timeout)
	defer cancel()

	own.executeStream(ctx, sess, cmd, handler)
}

func (own *Client) executeStream(ctx context.Context, session *ssh.Session, cmd *StreamCommand, handler StreamHandler) {
	// 1. 设置 stdin/stdout/stderr 缓冲区
	stdin, _ := session.StdinPipe()
	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()

	// 2. 初始化 PTY 伪终端
	if err := cmd.Terminal.Init(session); err != nil {
		_ = session.Close()
		atomic.AddInt64(&own.sessionTotal, -1)
		handler.CallOnError(wrapError(ErrSession, "init pty terminal: %w", err))
		return
	}
	own._debug("Init PTY terminal successfully")

	// 3. 启动命令会话
	if err := session.Start(cmd.Command); err != nil {
		_ = session.Close()
		atomic.AddInt64(&own.sessionTotal, -1)
		handler.CallOnError(wrapError(ErrSession, "start command: %w", err))
		return
	}
	own._debug("Start to execute session stream >>> %s", cmd.Command)

	// 4. 异步写入输入
	wg := &sync.WaitGroup{}
	if len(cmd.InputArgs) > 0 {
		wg.Go(func() {
			// 1. 输入参数
			if err := own.inputArguments(ctx, &Command{
				InputArgs:     cmd.InputArgs,
				InputInterval: cmd.InputInterval,
				stdin:         stdin,
			}, handler.CallOnStdin); err != nil {
				handler.CallOnError(err)
			}

			// 2. 发送 EOF (Ctrl+D) 通知输入结束，但不关闭管道
			if _, err := stdin.Write([]byte("\x04")); err != nil {
				own._debug("Failed to input EOF (Ctrl+D): %v", err)
			} else {
				own._debug("Input EOF (Ctrl+D)")
			}
		})
	} else {
		_ = stdin.Close() // 无输入时直接关闭 stdin，避免命令等待输入
	}

	// 读取输出
	wg.Go(func() {
		readStdoutStream(ctx, stdout, cmd.StdoutReadDelimit, cmd.StdoutReadTimeoutAfterClose, handler.CallOnError, handler.CallOnStdout)
		own._debug("Closed stdout pipe")
	})

	// 读取错误
	wg.Go(func() {
		readStderrStream(ctx, stderr, handler.CallOnError, handler.CallOnStderr)
		own._debug("Closed stderr pipe")
	})

	// 等待正常完成
	done := make(chan error, 1)
	closed := make(chan struct{}, 1)
	wg.Go(func() {
		defer atomic.AddInt64(&own.sessionTotal, -1)
		own.waitSession(session, closed, done)
		own._debug("Session done")
	})

	// 等待执行结束
	select {
	case err := <-done: // 执行正常结束
		handler.CallOnDone(err)
	case <-ctx.Done(): // 执行超时结束
		if err := own.shutdownSession(ctx, session, &Command{
			QuitTimeout: cmd.QuitTimeout,
			stdin:       stdin,
			doneChan:    done,
			closeChan:   closed,
		}); err != nil {
			handler.CallOnError(err)
		}
	}

	wg.Wait()
}

// readStdoutStream 读取输出流（返回原生的错误消息，不封装）
func readStdoutStream(ctx context.Context, stdout io.Reader, delimit byte, readTimeout time.Duration, handleError func(error), handleOutput func(string)) {
	// 创建可关闭的管道，用于中断阻塞读取
	pipeReader, pipeWriter := io.Pipe()
	defer func() { _ = pipeReader.Close() }()

	// 启动写入 goroutine：当 writer 关闭时 io.Copy 会返回
	go func() {
		_, err := io.Copy(pipeWriter, stdout)
		_ = pipeWriter.CloseWithError(err)
	}()

	// 启动读取 goroutine
	reader := bufio.NewReader(pipeReader)
	done := make(chan struct{})
	go func() {
		// 退出时通知主函数
		defer close(done)

		for {
			line, err := reader.ReadBytes(delimit)
			if len(line) > 0 {
				line = bytes.TrimSuffix(line, []byte{delimit})
				if delimit == '\n' {
					line = bytes.TrimSuffix(line, []byte{'\r'})
				}
				handleOutput(string(line))
			}

			if err != nil {
				if err != io.EOF && !errors.Is(err, io.ErrClosedPipe) {
					handleError(wrapError(ErrRead, "%w", err))
				}

				return
			}
		}
	}()

	// 等待读取结束或上下文取消
	select {
	case <-done:
		return
	case <-ctx.Done():
		// 上下文取消，关闭 pipe reader 以中断 io.Copy
		_ = pipeReader.Close()

		// 给 readTimeout 时间等待读取 goroutine 处理剩余数据
		timer := time.NewTimer(readTimeout)
		defer timer.Stop()

		// 等待结束或超时
		select {
		case <-done:
			return
		case <-timer.C:
			return
		}
	}
}

// readStderrStream 读取错误流（返回原生的错误消息，不封装）
func readStderrStream(ctx context.Context, stderr io.Reader, handleError func(error), handleOutput func(string)) {
	// 创建可关闭的管道，用于中断阻塞读取
	pipeReader, pipeWriter := io.Pipe()
	defer func() { _ = pipeReader.Close() }()

	// 启动写入 goroutine：当 writer 关闭时 io.Copy 会返回
	go func() {
		_, err := io.Copy(pipeWriter, stderr)
		_ = pipeWriter.CloseWithError(err)
	}()

	// 启动读取 goroutine
	reader := bufio.NewReader(pipeReader)
	done := make(chan struct{})
	go func() {
		// 退出时通知主函数
		defer close(done)

		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				line = bytes.TrimSuffix(line, []byte{'\n'})
				line = bytes.TrimSuffix(line, []byte{'\r'})
				handleOutput(string(line))
			}

			if err != nil {
				if err != io.EOF && !errors.Is(err, io.ErrClosedPipe) {
					handleError(wrapError(ErrRead, "%w", err))
				}

				return
			}
		}
	}()

	// 等待读取结束或上下文取消
	select {
	case <-done:
		return
	case <-ctx.Done():
		// 上下文取消，关闭 pipe reader 以中断 io.Copy
		_ = pipeReader.Close()

		// 给 readTimeout 时间等待读取 goroutine 处理剩余数据
		timer := time.NewTimer(time.Second)
		defer timer.Stop()

		// 等待结束或超时
		select {
		case <-done:
			return
		case <-timer.C:
			return
		}
	}
}
