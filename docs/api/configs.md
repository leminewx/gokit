# configs 配置文件 API
导入前缀：`gitee.com/leminewx/gokit/configs/...`

## configs/ini

INI / JSON 配置的加载、映射、修改与保存。读取支持文件、`io.Reader` 与字符串；
写入支持文件、`io.Writer` 与 `String()`。

```go
ini, err := ini.LoadFromINI("../../harness/testdata/test.ini")
if err != nil {
    return err
}

v, ok := ini.GetValue("detect", "type") // "1"
timeout, err := ini.GetDuration("server", "timeout") // "3s" → 3*time.Second
workers, err := ini.GetInt("server", "workers")
ini.SetValue("detect", "conf", "[0.5, 0.5]")
ini.RemoveKeys("stream", "timefile")
ini.RemoveSections("vroll")

if err := ini.Save("out.ini"); err != nil {
    return err
}
if _, err := ini.WriteTo(&buffer); err != nil { // 写入任意 io.Writer
    return err
}
text := ini.String()

// 映射到结构体
var cfg Config
if err := ini.Unmarshal(&cfg); err != nil {
    return err
}

// 结构体转配置
ini2, err := ini.Marshal(cfg)
```

| 符号 | 说明 |
|------|------|
| `LoadFromINI(filename string) (*INI, error)` | 读取 INI 文件，保留节与键的原始顺序 |
| `LoadFromReader(reader io.Reader) (*INI, error)` | 从任意读取器加载 |
| `LoadFromString(content string) (*INI, error)` | 从字符串加载 |
| `LoadFromJSON(filename string) (*INI, error)` | 读取 JSON 并转为同一模型 |
| `Marshal(v any) (*INI, error)` | 结构体 → `INI`（支持 `ini` / `json` 标签语义按实现） |
| `(*INI) Unmarshal(v any) error` | `INI` → 结构体 |
| `(*INI) GetValue(section, key string) (string, bool)` | 读单值 |
| `(*INI) SetValue(section, key, value string)` | 写单值 |
| `(*INI) RemoveKeys(section string, keys ...string)` | 删除键 |
| `(*INI) RemoveSections(sections ...string)` | 删除节 |
| `(*INI) Save(filename string) error` | 写回文件 |
| `(*INI) WriteTo(w io.Writer) (int64, error)` | 实现 `io.WriterTo` |
| `(*INI) String() string` | 序列化为 INI 文本 |
| `(*INI) MarshalJSON()` / `UnmarshalJSON([]byte)` | JSON 互操作，节名与键名排序，结果确定 |
| `INI` 字段 | `Sections []*Section`（有序，全局节在首位）、`SectionMap map[string]*Section`（快速查找） |
| `Section` 字段 | `Name`（空串表示全局节）、`Keys map[string]string`、`Order []string` |

### 类型化访问

| 方法 | 说明 |
|------|------|
| `GetString(section, key) string` | 缺失时返回空串，不报错 |
| `GetInt` / `GetInt64` / `GetUint64` / `GetFloat64` / `GetBool` | 转换失败返回 `ErrConvertFailed`，键不存在返回 `ErrKeyNotFound` |
| `GetDuration(section, key) (time.Duration, error)` | 支持 `300ms` / `1m30s` 或整数纳秒 |
| `GetStrings(section, key) ([]string, error)` | 支持 JSON 数组 `["a","b"]` 与逗号分隔 `a,b` |
| `HasSection` / `HasKey` / `GetSectionNames` / `GetKeys` / `GetSortedSectionNames` | 存在性判断、节名与键名枚举 |

结构体映射规则：

- 字段标签 `ini:"section.key"` 指定节与键；非结构体字段映射到全局节。
- `time.Duration`、`time.Time` 与实现 `encoding.TextUnmarshaler` 的类型按标量处理，
  其余结构体视为「节」。
- 切片字段优先按 JSON 数组解析，失败再按逗号分隔；空值得到空切片。
- 行内注释仅在 `#` / `;` 前有空白且不在引号内时生效，URL 等含 `#` 的值不会被截断。

测试夹具固定在 `harness/testdata/test.ini`，用例通过相对路径
`../../harness/testdata/test.ini` 读取；不要在其他位置新建 `testdata` 目录。