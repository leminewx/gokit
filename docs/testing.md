# 测试指南

## 测试分层

| 层级 | 触发方式 | 依赖 | 典型包 |
|------|----------|------|--------|
| 单元测试 | 默认参与构建 | 无外部依赖，可用回环/伪终端 | 绝大多数包 |
| 集成测试 | `-tags integration` | 真实设备、外部服务、指定网卡 | `nets`、`nets/tcp`、`nets/serial`、`nets/smtpx`、`nets/sshx` |

默认 `go test ./...` 必须在无网络、无设备的机器上全绿；任何依赖真实环境的用例都必须用
`//go:build integration` 隔离，并在文件头注释中写明依赖与执行方式。

## 测试数据规范

全仓唯一的测试数据目录是 **`harness/testdata`**。

```
gokit/
└── harness/
    └── testdata/
        └── test.ini     # configs/ini 用例的固定输入
```

规则：

1. **禁止**在包目录、仓库根新建 `testdata/`；历史遗留的根目录 `testdata/` 已移除。
2. 读取固定数据：使用相对路径 `../../harness/testdata/<file>`，例如
   `configs/ini/ini_test.go`：
   ```go
   ini, err := LoadFromINI("../../harness/testdata/test.ini")
   ```
3. 需要在测试过程中落盘的数据（如 RSA 密钥）也写入 `harness/testdata`，
   并在 `t.Cleanup` 中删除，保证工作区干净。参考 `cryptos/rsax/rsax_test.go`：
   ```go
   const _TESTDATA_DIR = "../../harness/testdata"

   func testdataFile(t *testing.T, name string) string {
       t.Helper()
       if err := os.MkdirAll(_TESTDATA_DIR, 0o755); err != nil {
           t.Fatalf("创建测试数据目录失败: %v", err)
       }
       path := filepath.Join(_TESTDATA_DIR, name)
       t.Cleanup(func() { _ = os.Remove(path) })
       return path
   }
   ```
4. 与仓库无关的一次性临时文件（流式读写的临时文件等）使用 `t.TempDir()` 或
   `os.CreateTemp("", ...)`，不要落到仓库目录。
5. `harness/testdata/test.ini` 是固定夹具，可以提交；用例生成的 `*.pem` 等中间产物不提交。

## 集成测试清单

所有集成用例通过环境变量注入设备信息，变量缺失时自动 `t.Skip`，不会因缺少设备而失败。

| 文件 | 用例 | 依赖 |
|------|------|------|
| `nets/net_env_test.go` | `TestGetPublicIP` | `GOKIT_NET_INTERFACE`（本机网卡名）与 `GOKIT_NET_PROBE_ADDR`（默认 `8.8.8.8:80`） |
| `nets/tcp/client_integration_test.go` | `TestClientIntegration` | `GOKIT_TCP_HOST` / `GOKIT_TCP_PORT`（默认 50700） |
| `nets/serial/port_integration_test.go` | `TestPortLoopback` | `GOKIT_SERIAL_PORT`（如 `COM6` / `/dev/ttyUSB0`） |
| `nets/smtpx/client_integration_test.go` | `TestClientSendText` / `TestClientSendHTML` | `SMTPX_HOST` / `SMTPX_USER` / `SMTPX_PASS`，可选 `SMTPX_PORT` / `SMTPX_TO` |
| `nets/sshx/integration_test.go` | `testClient` 辅助函数 | `GOKIT_SSH_HOST` / `GOKIT_SSH_USER` / `GOKIT_SSH_PASS` |
| `nets/sshx/session_test.go` | 会话与命令执行 | 同上 |
| `nets/sshx/stream_test.go` | 流式执行 | 同上 |
| `nets/sshx/sshx_live_test.go` | 真实设备端到端 | 同上 |
| `nets/sshx/command/collect_live_test.go` | `TestCollectLive` | `GOKIT_SSH_HOST` / `GOKIT_SSH_USER` / `GOKIT_SSH_PASSWORD` |

> 密码变量 `GOKIT_SSH_PASSWORD` 与 `GOKIT_SSH_PASS` 等价，`nets/sshx` 与 `nets/sshx/command`
> 两处用例都会依次读取，设置其中一个即可。

`nets/sshx` 的多个测试文件共享 `integration_test.go` 中的 `testClient` helper，因此**必须整体**
带 `integration` 标签，否则默认构建会因缺少 helper 而失败。`nets/sshx/command` 的解析器用例
不依赖外部环境，默认参与构建；仅 `TestCollectLive` 需要真实设备，未设置 `GOKIT_SSH_HOST`
时自动跳过。可选环境变量：`GOKIT_SSH_PORT`（默认 22）、`GOKIT_SSH_INTERVAL`（默认 2s）。

TCP 的单元用例在 `nets/tcp/client_test.go`，通过本地回环监听模拟服务端，默认参与测试；
`client_integration_test.go` 只覆盖真实设备。


## 常用命令

```bash
# 编译与静态检查（多包时串行，避免 Windows 下并发链接偶发失败）
go build -p 1 ./...
go vet -p 1 ./...

# 默认全量测试（含竞态检测）
go test -race -count=1 -p 1 -timeout 900s ./...

# 单个包
go test -race -count=1 ./nets/mesh

# 只跑某个用例
go test -race -count=1 -run TestMesh_E2E_FullLifecycle ./nets/mesh

# 集成测试（变量缺失时自动跳过）
$env:GOKIT_TCP_HOST="192.0.2.30"; $env:GOKIT_TCP_PORT="50700"
go test -tags integration -count=1 -run TestClientIntegration -v ./nets/tcp

$env:GOKIT_SSH_HOST="192.0.2.10"; $env:GOKIT_SSH_USER="root"; $env:GOKIT_SSH_PASS="******"
go test -tags integration -count=1 ./nets/sshx

# 采集器实机验证
go test -tags integration -count=1 -run TestCollectLive -v ./nets/sshx/command

# 集成测试也要保证能编译
go vet -tags integration ./nets ./nets/tcp ./nets/serial ./nets/smtpx ./nets/sshx ./nets/sshx/command
```

Windows PowerShell 不支持 `&&`，多条命令用 `;` 分隔。

## Makefile 与 CI

仓库根目录提供 `Makefile` 作为统一入口，本地与 CI 使用同一套命令：

| 目标 | 作用 |
|------|------|
| `make fmt` | `gofmt -w .`，同时统一为 LF 行尾 |
| `make fmt-check` | 检查未格式化文件，非空即失败（CI 使用） |
| `make vet` | `go vet -p 1 ./...` |
| `make build` | `go build -p 1 ./...` |
| `make test` | `go test -count=1 -p 1 ./...` |
| `make race` | `go test -count=1 -p 1 -race ./...` |
| `make cover` | 生成 `coverage.out` 并输出函数级覆盖率 |
| `make lint` | `golangci-lint run`（配置见 `.golangci.yml`） |
| `make check` | `fmt-check` + `vet` + `build` + `test`，提交前建议执行 |

CI 建议按三个 job 组织：`build`（格式、vet、build、测试）、`race`、`lint`（golangci-lint v2），
直接调用上表中的 `make` 目标即可。Go 版本统一从 `go.mod` 读取，vet / build / test 均使用 `-p 1`
以降低并行构建的内存峰值。
`golangci-lint` 未安装时：

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

## 常见问题

- **`go test` 偶发「go.exe 无法运行: 拒绝访问」**：杀毒软件或上次进程残留导致，重试即可。
- **中文日志在 PowerShell 中乱码**：将输出重定向到文件时显式指定编码：
  ```powershell
  go test ./nets/mesh 2>&1 | Out-File -Encoding utf8 test.log
  Select-String -Path test.log -Pattern 'PASS:|FAIL:'
  ```
- **`gofmt -l` 列出所有文件但 `gofmt -d` 无差异**：多为 CRLF 行尾导致，执行
  `gofmt -w <dir>` 统一后即可。
- **多包测试超时或端口冲突**：加 `-p 1` 串行执行；Mesh 用例使用随机端口，避免与本地服务冲突。
- **依赖系统时钟的偶发失败**：Windows 系统时钟粒度较粗，连续调用 `time.Now().UnixNano()`
  可能得到相同值。库内已改为单调计数器生成消息序列号；新增依赖时间或定时器的代码应通过
  `gokit/timex` 注入时间源，测试使用 `timex.NewManualAt()` + `Advance`，参考
  `cache` 的用例，不要使用 `time.Sleep` 等待过期。
