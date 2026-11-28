// Package command 提供 Linux 设备通用命令与 /proc 伪文件输出的解析与采集能力，
// 用于在不安装任何代理的前提下，通过 SSH 获取整机性能指标。
//
// 设计目标：
//   - 只依赖 POSIX shell 与 /proc，可同时适配 GNU coreutils（服务器）、BusyBox（嵌入式）
//     与 Android/ToyBox 设备；
//   - 解析函数只处理文本，不依赖 SSH 连接，便于单元测试与历史数据回放；
//   - 采集入口 Collector 在单个会话内分段输出，减少往返开销与超时风险。
//
// 典型用法：
//
//	client, err := sshx.NewClient("10.0.0.10", "root", "******")
//	if err != nil {
//		return err
//	}
//	defer func() { _ = client.Close() }()
//
//	snapshot, err := command.NewCollector(client).Collect(context.Background())
//	if err != nil {
//		return err
//	}
//
//	fmt.Println(snapshot.Format())   // 人类可读的整机概览
//	fmt.Println(snapshot.CPU.Used)   // 结构化字段，可直接序列化为 JSON
//
// 也可以单独使用解析函数处理本地文件或历史采集结果：
//
//	stat, err := command.ParseProcStat(text)   // /proc/stat
//	mem, err := command.ParseMemInfo(text)     // /proc/meminfo
//	fs, err := command.ParseDf(text)           // df -P -k
//	procs, err := command.ParsePS(text)        // ps aux / ps -eo
//	top, err := command.ParseTop(rawTopOutput) // top -b -n 1
//
// 解析函数兼容不同实现的常见差异：列宽换行（BusyBox df）、缺失 MemAvailable
// （内核 3.14 之前）、MiB/GiB 自动换单位（GNU top）、%CPU/%MEM 列缺失（BusyBox ps）、
// `top -1` 的逐核 CPU 行（按各核平均值近似整机）。采集脚本会导出 LC_ALL=C，
// 避免本地化后的 df/top 表头导致解析失败。无法识别的输入统一返回以 ErrParse 包装的错误。
package command
