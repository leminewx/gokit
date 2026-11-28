package stream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"

	"gitee.com/leminewx/gokit/timex"
)

// NewStreamFromLineFile 将指定文件按行处理后，生成一个新的数据流
//   - 数据读取完毕后，会自动关闭 reader
//   - 取消信号触发后立即停止读取并关闭 reader
func NewStreamFromLineFile[T any](ctx context.Context, reader io.ReadCloser, format func(line []byte, err error) (T, bool)) Stream[T] {
	if ctx == nil {
		ctx = context.Background()
	}
	if reader == nil || format == nil {
		return newEmptyStream[T]()
	}

	output := make(chan Item[T], getCacheSize())
	own := &stream[T]{source: output, ctx: ctx, clock: timex.NewSystemClock()}
	bufReader := bufio.NewReader(reader)
	go func() {
		defer func() {
			_ = reader.Close()
			close(output)
		}()

		var index uint64
		for {
			if ctx.Err() != nil {
				return
			}

			line, err := bufReader.ReadBytes('\n')
			if len(line) > 0 {
				line = bytes.TrimSuffix(line, []byte("\n"))
				line = bytes.TrimSuffix(line, []byte("\r"))
			}

			if value, ok := format(bytes.TrimSpace(line), err); ok {
				item := NewItem(value)
				item.SetIndex(index)
				index++

				if !send(ctx, output, item) {
					return
				}
			}

			if err == io.EOF {
				return
			}
		}
	}()

	return own
}

// NewStreamFromJsonFile 将指定json文件处理后，生成一个新的数据流
//   - 数据格式必须是列表对象：[]object
//   - 数据读取完毕后，会自动关闭 reader
//   - 取消信号触发后立即停止读取并关闭 reader
func NewStreamFromJsonFile[T any](ctx context.Context, reader io.ReadCloser, format func(data T, err error) (T, bool), newer ...func() T) Stream[T] {
	if ctx == nil {
		ctx = context.Background()
	}
	if reader == nil || format == nil {
		return newEmptyStream[T]()
	}

	var newFn func() T
	if len(newer) > 0 && newer[0] != nil {
		newFn = newer[0]
	}

	// 读取第一个 token，必须是 [
	// 若文件不是 [ 开头，则直接返回空流
	jsonDecoder := json.NewDecoder(reader)
	if tok, err := jsonDecoder.Token(); err != nil {
		_ = reader.Close()
		return newEmptyStream[T]()
	} else if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		_ = reader.Close()
		return newEmptyStream[T]()
	}

	output := make(chan Item[T], getCacheSize())
	own := &stream[T]{source: output, ctx: ctx, clock: timex.NewSystemClock()}
	go func() {
		defer func() {
			_ = reader.Close()
			close(output)
		}()

		var index uint64
		for jsonDecoder.More() {
			if ctx.Err() != nil {
				return
			}

			var data T
			if newFn != nil {
				data = newFn()
			}

			if value, ok := format(data, jsonDecoder.Decode(&data)); ok {
				item := NewItem(value)
				item.SetIndex(index)
				index++

				if !send(ctx, output, item) {
					return
				}
			}
		}
		_, _ = jsonDecoder.Token()
	}()

	return own
}
