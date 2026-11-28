package ini

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Save 将配置保存到 ini 文件
func (own *INI) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return wrapError(ErrWriteFailed, "%q: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	if _, err := own.WriteTo(file); err != nil {
		return wrapError(ErrWriteFailed, "%q: %w", filename, err)
	}

	return nil
}

// WriteTo 将配置以 ini 文本写入 w，实现 io.WriterTo
func (own *INI) WriteTo(writer io.Writer) (int64, error) {
	counter := &countingWriter{writer: writer}
	if err := own.writeTo(counter); err != nil {
		return counter.written, err
	}

	return counter.written, nil
}

// String 返回配置的 ini 文本表示
func (own *INI) String() string {
	var builder strings.Builder
	_ = own.writeTo(&builder)

	return builder.String()
}

// writeTo 按节的顺序输出文本：全局节无节头，节之间以空行分隔
func (own *INI) writeTo(output io.Writer) error {
	writer := bufio.NewWriter(output)
	for idx, section := range own.Sections {
		if section.Name == "" {
			// 全局节：直接输出键值对，无节头
			for _, key := range section.Order {
				if _, err := fmt.Fprintf(writer, "%s = %s\n", key, section.Keys[key]); err != nil {
					return err
				}
			}

			if len(section.Order) > 0 && idx < len(own.Sections)-1 {
				if _, err := fmt.Fprintln(writer); err != nil {
					return err
				}
			}

			continue
		}

		if _, err := fmt.Fprintf(writer, "[%s]\n", section.Name); err != nil {
			return err
		}

		for _, key := range section.Order {
			if _, err := fmt.Fprintf(writer, "%s = %s\n", key, section.Keys[key]); err != nil {
				return err
			}
		}

		// 节之间空行
		if idx < len(own.Sections)-1 {
			if _, err := fmt.Fprintln(writer); err != nil {
				return err
			}
		}
	}

	return writer.Flush()
}

// countingWriter 统计已写入的字节数
type countingWriter struct {
	writer  io.Writer
	written int64
}

// Write 写入并累计字节数
func (own *countingWriter) Write(data []byte) (int, error) {
	n, err := own.writer.Write(data)
	own.written += int64(n)

	return n, err
}
