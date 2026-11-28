package stream

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestWriteToJSON_Basic 验证正常写入
func TestWriteToJSON_Basic(t *testing.T) {
	buf := &bytes.Buffer{}
	stream := NewStreamOf(1, 2, 3, 4, 5)
	newStream := stream.WriteToJSON(buf)
	newStream.Wait() // 等待后台写入完成

	var result []int
	err := json.Unmarshal(buf.Bytes(), &result)
	assertNoError(t, err)
	assertEqualIntSlice(t, []int{1, 2, 3, 4, 5}, result)
}

// TestWriteToJSON_Empty 验证空流输出空数组
func TestWriteToJSON_Empty(t *testing.T) {
	buf := &bytes.Buffer{}
	stream := NewStreamOf[int]()
	newStream := stream.WriteToJSON(buf)
	newStream.Wait()

	var result []int
	err := json.Unmarshal(buf.Bytes(), &result)
	assertNoError(t, err)
	assertLen(t, result, 0)
}

// TestWriteToJSON_EncodeError 验证不可编码元素被跳过，JSON 格式仍然合法
func TestWriteToJSON_EncodeError(t *testing.T) {
	buf := &bytes.Buffer{}
	// 构造包含不可编码元素（func）的流
	stream := NewStreamOf(
		map[string]interface{}{"a": 1},
		map[string]interface{}{"b": func() {}}, // 不可编码
		map[string]interface{}{"c": 3},
	)
	newStream := stream.WriteToJSON(buf)
	newStream.Wait()

	var result []map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	assertNoError(t, err)
	if len(result) != 2 {
		t.Errorf("期望 2 个元素, 实际 %d", len(result))
	}
	if len(result) >= 2 {
		if v, ok := result[0]["a"]; !ok || v != float64(1) {
			t.Errorf("第一个元素不正确")
		}
		if v, ok := result[1]["c"]; !ok || v != float64(3) {
			t.Errorf("第二个元素不正确")
		}
	}
}

// TestWriteToJSON_WriteError 验证写入错误时，JSON 格式仍保持合法
func TestWriteToJSON_WriteError(t *testing.T) {
	buf := &bytes.Buffer{}
	fw := &failWriter{
		Writer:    buf,
		failAfter: 2, // 第二次 Write（即 "[\n" 之后）失败
	}
	stream := NewStreamOf(1, 2, 3, 4, 5)
	newStream := stream.WriteToJSON(fw)
	newStream.Wait()

	data := buf.Bytes()
	if len(data) == 0 {
		t.Skip("写入完全失败，跳过验证")
	}
	var result []int
	err := json.Unmarshal(data, &result)
	if err != nil {
		t.Errorf("JSON 格式非法: %v, 内容: %s", err, string(data))
	} else {
		t.Logf("成功写入的元素: %v", result)
	}
}

// TestWriteToJSON_Chain 验证 WriteToJSON 返回的新流可以继续链式操作
func TestWriteToJSON_Chain(t *testing.T) {
	buf := &bytes.Buffer{}
	stream := NewStreamOf(1, 2, 3, 4, 5)
	newStream := stream.WriteToJSON(buf)

	// 对返回的流进行 Map
	mapped := Map(newStream, func(item Item[int]) int {
		return item.GetValue() * 2
	})

	var results []int
	mapped.Consume(func(item Item[int]) {
		results = append(results, item.GetValue())
	})
	mapped.Wait()

	// 验证映射结果
	assertEqualIntSlice(t, []int{2, 4, 6, 8, 10}, results)

	// 验证 JSON 内容为原始数据
	var jsonData []int
	err := json.Unmarshal(buf.Bytes(), &jsonData)
	assertNoError(t, err)
	assertEqualIntSlice(t, []int{1, 2, 3, 4, 5}, jsonData)
}

// TestWriteToJSON_LargeData 大数据量测试（可用 -short 跳过）
func TestWriteToJSON_LargeData(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过大数据测试")
	}
	buf := &bytes.Buffer{}
	const count = 10000
	data := make([]int, count)
	for i := 0; i < count; i++ {
		data[i] = i
	}
	stream := NewStreamOf(data...)
	newStream := stream.WriteToJSON(buf)
	newStream.Wait()

	var result []int
	err := json.Unmarshal(buf.Bytes(), &result)
	assertNoError(t, err)
	if len(result) != count {
		t.Errorf("期望 %d 个元素, 实际 %d", count, len(result))
	}
	if len(result) > 0 {
		if result[0] != 0 || result[count-1] != count-1 {
			t.Error("数据内容不正确")
		}
	}
}

// TestWriteToJSON_NilWriter 验证 writer 为 nil 时的行为
func TestWriteToJSON_NilWriter(t *testing.T) {
	stream := NewStreamOf(1, 2, 3)
	newStream := stream.WriteToJSON(nil)
	if newStream == nil {
		t.Error("期望返回非空流")
	}
	// 确保不会 panic
	newStream.Wait()
}
