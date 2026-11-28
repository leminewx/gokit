package stream

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"testing"
)

func TestNewStreamFromLineFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_lines")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	content := "hello\nworld\n123"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tmpFile.Close() }()

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()

	ctx := context.Background()
	s := NewStreamFromLineFile(ctx, file, func(line []byte, _ error) ([]byte, bool) {
		return line, true
	})
	got := collect(s)
	want := [][]byte{[]byte("hello"), []byte("world"), []byte("123")}
	for i := range got {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("NewStreamFromLineFile = %v, want %v", got[i], want[i])
		}
	}
}

func TestNewStreamFromJsonFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	content := `[{"name":"a","val":1},{"name":"b","val":2}]`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tmpFile.Close() }()

	type Data struct {
		Name string
		Val  int
	}
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()

	ctx := context.Background()
	s := NewStreamFromJsonFile(ctx, file, func(data Data, err error) (Data, bool) {
		if err != nil {
			return data, false
		}
		return data, true
	}, func() Data { return Data{} })
	got := collect(s)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Name != "a" || got[0].Val != 1 {
		t.Errorf("first item = %+v, want {a 1}", got[0])
	}
	if got[1].Name != "b" || got[1].Val != 2 {
		t.Errorf("second item = %+v, want {b 2}", got[1])
	}
}

// countingReader 持续产出文本行的只读资源，用于验证取消后 reader 是否被关闭
type countingReader struct {
	lock   sync.Mutex
	lines  int
	read   int
	closed chan struct{}
	once   sync.Once
}

// newCountingReader 构造一个可产出指定行数的 countingReader
func newCountingReader(lines int) *countingReader {
	return &countingReader{lines: lines, closed: make(chan struct{})}
}

// Read 按行产出文本内容，读完后返回 io.EOF
func (own *countingReader) Read(p []byte) (int, error) {
	own.lock.Lock()
	defer own.lock.Unlock()

	if own.read >= own.lines {
		return 0, io.EOF
	}

	own.read++
	line := fmt.Sprintf("line-%d\n", own.read)

	return copy(p, line), nil
}

// Close 关闭资源，可重复调用
func (own *countingReader) Close() error {
	own.once.Do(func() { close(own.closed) })

	return nil
}
