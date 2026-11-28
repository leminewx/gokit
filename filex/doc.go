// Package filex 提供文件的原子写入、原子复制与路径定位。
//
// WriteFileAtomic 先写同目录临时文件、fsync 后再 rename，CopyFileAtomic 复用同一
// 流程，因此目标文件要么保持完整旧内容、要么切换到完整新内容，不会出现半截文件；
// 任一失败路径都会清理临时文件。临时文件与目标同目录，不涉及跨文件系统 rename。
//
// GetModuleRoot 从工作目录（或本包源码位置）逐级向上定位 go.mod，ResolvePath 把相对
// 路径限制在给定根目录内并拒绝 ".." 越界，二者可组合定位仓库内的数据目录、静态资源
// 或下载目录，无需依赖测试运行方式。
//
// 包内不递归创建父目录，也不跟随符号链接、不保留稀疏文件空洞；WriteFileAtomic 的
// 权限由调用方显式指定，传 0 时按 0o600 处理，CopyFileAtomic 继承源文件权限。
package filex
