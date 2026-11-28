//go:build windows

package progressbar

import (
	"os"

	"golang.org/x/sys/windows"
)

// getTerminalSize 获取给定文件对应的控制台宽度（列数）
//   - f 为 nil 或非控制台（重定向到文件/管道、io.Discard 等）时，
//     直接返回默认宽度，避免无意义的系统调用
func (own *Progressbar) getTerminalSize(f *os.File) int {
	if f == nil {
		return _DEFAULT_TERMINAL_SIZE
	}

	handle := windows.Handle(f.Fd())

	// 判断句柄是否为控制台：GetConsoleMode 对非控制台句柄必然失败，
	// 此时无需继续查询，直接回落默认宽度
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return _DEFAULT_TERMINAL_SIZE
	}

	var consoleInfo windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(handle, &consoleInfo); err != nil {
		return _DEFAULT_TERMINAL_SIZE
	}

	// 可视窗口宽度（左闭右闭区间）
	width := int(consoleInfo.Window.Right - consoleInfo.Window.Left + 1)
	if width <= 0 {
		return _DEFAULT_TERMINAL_SIZE
	}

	return width
}
