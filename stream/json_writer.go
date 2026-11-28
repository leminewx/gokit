package stream

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
)

// WriteToJSON 写入到 JSON 写入器
//   - writer: JSON 写入器
//   - 注意: 输出格式为 [obj1, obj2, ...]
//   - 注意: 这是旁路操作，写入失败也不影响原始数据的流出
func (own *stream[T]) WriteToJSON(writer io.Writer) Stream[T] {
	if writer == nil {
		return own
	}

	// 写入数组开头
	bufWriter := bufio.NewWriter(writer)
	if _, err := bufWriter.Write([]byte("[\n")); err != nil {
		slog.Error("failed to write json array start", "error", err)
		return own // 返回原始流，原流未被消费，调用者仍可继续使用
	}

	ctx := own.ctx
	output := make(chan Item[T], getCacheSize())
	go func() {
		defer func() {
			// 写入数组结尾并冲刷缓冲
			if _, err := bufWriter.Write([]byte("\n]")); err != nil {
				slog.Error("failed to write json array end", "error", err)
			}
			if err := bufWriter.Flush(); err != nil {
				slog.Error("failed to flush json writer", "error", err)
			}

			// 最后关闭 output，通知下游数据已全部写入
			close(output)
		}()

		var (
			needComma bool
			chunk     bytes.Buffer
			encoded   bytes.Buffer
		)
		encoder := json.NewEncoder(&encoded)

		for {
			item, ok := recv(ctx, own.source)
			if !ok {
				return
			}

			// 1. 转发数据到新流（独立于写入）
			if !send(ctx, output, item) {
				return
			}

			// 2. 编码当前元素到临时 buffer
			encoded.Reset()
			if err := encoder.Encode(item.GetValue()); err != nil {
				slog.Error("failed to encode json value", "error", err, "index", item.GetIndex())
				continue // 编码失败，跳过该元素，不写入文件
			}

			// 3. 构建本次要写入的完整片段（逗号 + 编码内容）
			//   - Encode 会在末尾追加换行符，因此无需额外添加
			chunk.Reset()
			if needComma {
				chunk.WriteString(",\n")
			}
			chunk.Write(encoded.Bytes())

			// 4. 原子写入 chunk；写入失败时保持 needComma 不变（该元素未成功写入）
			if _, err := bufWriter.Write(chunk.Bytes()); err != nil {
				slog.Error("failed to write json chunk", "error", err, "index", item.GetIndex())
				continue
			}

			// 5. 只有成功写入后，才标记需要逗号
			needComma = true
		}
	}()

	return own.wrap(output)
}
