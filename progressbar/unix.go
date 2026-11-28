//go:build !windows

package progressbar

import (
	"os"

	"golang.org/x/sys/unix"
)

// getTerminalSize 获取给定文件对应的终端宽度（列数）
//   - f 为 nil 或非终端（重定向到文件/管道、io.Discard 等）时，
//     直接返回默认宽度，避免无意义的系统调用
func (own *Progressbar) getTerminalSize(f *os.File) int {
	if f == nil {
		return _DEFAULT_TERMINAL_SIZE
	}

	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return _DEFAULT_TERMINAL_SIZE
	}

	// 部分伪终端/非交互环境会返回 0，需回落默认宽度
	width := int(ws.Col)
	if width <= 0 {
		return _DEFAULT_TERMINAL_SIZE
	}

	return width
}
