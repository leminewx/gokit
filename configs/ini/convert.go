package ini

import (
	"encoding"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	// durationType 用于识别 time.Duration（底层为 int64，但语义不同）
	durationType = reflect.TypeOf(time.Duration(0))
	// textUnmarshalerType 用于识别自定义文本解析类型（如 time.Time）
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// isSectionStruct 判断字段是否应视为独立节
//   - 普通结构体视为节；time.Time 与实现 TextUnmarshaler 的类型按标量处理
func isSectionStruct(field reflect.Value) bool {
	if field.Kind() != reflect.Struct {
		return false
	}

	typ := field.Type()
	if typ == reflect.TypeOf(time.Time{}) {
		return false
	}

	return !reflect.PointerTo(typ).Implements(textUnmarshalerType)
}

// setField 将字符串 val 写入反射字段 f
//   - 支持基础类型、切片、指针、time.Duration 与 encoding.TextUnmarshaler
func setField(f reflect.Value, val string) error {
	if f.Kind() == reflect.Ptr {
		if strings.TrimSpace(val) == "" {
			return nil
		}

		target := reflect.New(f.Type().Elem())
		if err := setField(target.Elem(), val); err != nil {
			return err
		}

		f.Set(target)
		return nil
	}

	if f.CanAddr() {
		if unmarshaler, ok := f.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(val))
		}
	}

	if f.Type() == durationType {
		duration, err := parseDuration(val)
		if err != nil {
			return err
		}

		f.SetInt(int64(duration))
		return nil
	}

	switch f.Kind() {
	case reflect.String:
		f.SetString(unquote(val))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err != nil {
			return err
		}
		f.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 64)
		if err != nil {
			return err
		}
		f.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return err
		}
		f.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(strings.TrimSpace(val))
		if err != nil {
			return err
		}
		f.SetBool(b)
	case reflect.Slice:
		return setSlice(f, val)
	default:
		return wrapError(ErrUnsupportedType, "%s", f.Kind())
	}

	return nil
}

// setSlice 解析切片字段：优先 JSON 数组，其次逗号分隔
func setSlice(f reflect.Value, val string) error {
	val = strings.TrimSpace(val)
	if val == "" {
		f.Set(reflect.MakeSlice(f.Type(), 0, 0))
		return nil
	}

	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		ptr := reflect.New(f.Type())
		if err := json.Unmarshal([]byte(val), ptr.Interface()); err == nil {
			f.Set(ptr.Elem())
			return nil
		}
	}

	parts := strings.Split(val, ",")
	newSlice := reflect.MakeSlice(f.Type(), len(parts), len(parts))
	for i, part := range parts {
		value, err := convertToValue(part, f.Type().Elem())
		if err != nil {
			return wrapError(ErrConvertFailed, "element %d: %w", i, err)
		}
		newSlice.Index(i).Set(value)
	}

	f.Set(newSlice)
	return nil
}

// convertToValue 将字符串转换为指定类型
func convertToValue(str string, typ reflect.Type) (reflect.Value, error) {
	ptr := reflect.New(typ)
	if err := setField(ptr.Elem(), str); err != nil {
		return reflect.Value{}, err
	}

	return ptr.Elem(), nil
}

// parseDuration 解析时长：优先 Go 时长格式（如 30s），其次纯数字按纳秒处理
func parseDuration(val string) (time.Duration, error) {
	val = strings.TrimSpace(val)
	if duration, err := time.ParseDuration(val); err == nil {
		return duration, nil
	}

	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return time.Duration(n), nil
}

// unquote 去掉成对包裹的单/双引号
func unquote(val string) string {
	val = strings.TrimSpace(val)
	if len(val) < 2 {
		return val
	}

	if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
		return val[1 : len(val)-1]
	}

	return val
}
