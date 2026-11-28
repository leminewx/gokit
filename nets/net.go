package nets

import (
	"errors"
	"net"
	"sort"
	"strings"

	"gitee.com/leminewx/gokit"
)

// _DEFAULT_MAX_IPV4_COUNT 生成 IP 列表时默认允许的最大地址数量
//   - 单次生成百万级地址（如 /8 的 1600 万个字符串）会瞬间吃掉数百 MB 内存，
//     因此对默认上限做约束；确有大网段需求时使用 GenerateIPV4sLimit 显式放宽
const _DEFAULT_MAX_IPV4_COUNT = 1 << 16

// connectionErrors 常见断开错误短语表
//   - 注意：超时（i/o timeout）不在其中，超时是暂时性错误，不代表连接已断开
var connectionErrors = []string{
	"eof",                              // 对端关闭
	"broken pipe",                      // 管道已损坏
	"connection reset by peer",         // 连接被重置
	"no route to host",                 // 无法到达主机
	"connection reset",                 // 连接被重置
	"connection refused",               // 连接被拒绝
	"network is unreachable",           // 网络不可达
	"use of closed network connection", // 连接已关闭
	"ssh: connection aborted",          // ssh: 连接被中止
	"ssh: handshake failed",            // ssh: 握手失败
	"ssh: connection lost",             // ssh: 连接丢失
	"ssh: disconnected",                // ssh: 断开
}

// IsConnectionError 判断错误是否为典型的网络连接断开错误
//   - 超时类错误不算断连，调用方应重试而非重建连接
//   - DNS 解析错误（未找到主机、解析器超时等）不算断连，属于配置或解析器问题
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// DNS 解析错误同时实现了 net.Error，必须先排除，否则会被下面的 net.Error 分支误判
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return false
	}

	// 如果实现了 net.Error 且不是超时，视为连接错误
	var netErr net.Error
	if errors.As(err, &netErr) && !netErr.Timeout() {
		return true
	}

	// 常见断开错误短语
	errStr := strings.ToLower(err.Error())
	for _, phrase := range connectionErrors {
		if strings.Contains(errStr, phrase) {
			return true
		}
	}

	return false
}

// GenerateIPV4s 根据 CIDR 生成网段内所有可用 IPV4 地址列表
//   - cidr: 例如 "192.168.1.0/24"
//   - skipNetworkBroadcast: 是否跳过网络地址（第一个）和广播地址（最后一个）
//
// 注意：
//   - 对于 /31 网段 (RFC 3021)，无网络/广播地址，skipNetworkBroadcast 无效，返回两个可用 IP
//   - 对于 /32 网段，返回仅有的一个 IP，skipNetworkBroadcast 无效
func GenerateIPV4s(cidr string, skipNetworkBroadcast bool) ([]string, error) {
	return GenerateIPV4sLimit(cidr, skipNetworkBroadcast, _DEFAULT_MAX_IPV4_COUNT)
}

// GenerateIPV4sLimit 同 GenerateIPV4s，但允许自定义地址数量上限
//   - maxCount <= 0 时使用默认上限 _DEFAULT_MAX_IPV4_COUNT
//   - 网段地址总数超过 maxCount 时返回 ErrTooManyAddresses，避免无上限分配导致 OOM
func GenerateIPV4sLimit(cidr string, skipNetworkBroadcast bool, maxCount int) ([]string, error) {
	if maxCount <= 0 {
		maxCount = _DEFAULT_MAX_IPV4_COUNT
	}

	// 1. 解析 CIDR
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, wrapError(ErrInvalidCIDR, "%q: %w", cidr, err)
	}

	ip := ipNet.IP.To4()
	if ip == nil {
		return nil, wrapError(ErrIPv4Only, "%q", cidr)
	}

	mask := ipNet.Mask
	network := make(net.IP, 4)
	copy(network, ipNet.IP) // 网络地址

	// 2. 计算广播地址（网络地址 | ^掩码）
	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = network[i] | ^mask[i]
	}

	// 3. 获取网段大小（地址总数）
	ones, bits := mask.Size()
	if bits != 32 {
		return nil, wrapError(ErrInvalidMask, "%q has %d bit mask", cidr, bits)
	}
	// 用 int64 计算：32 位平台上 1<<32 会溢出为 0，导致 /0 等超大网段绕过上限校验
	total := int64(1) << (32 - ones) // 地址总数
	if total > int64(maxCount) {
		return nil, wrapError(ErrTooManyAddresses, "%q has %d addresses, limit %d", cidr, total, maxCount)
	}

	// 特殊情况：/32 和 /31
	if total == 1 { // /32
		return []string{network.String()}, nil
	}
	if total == 2 { // /31
		// RFC 3021：两个地址都是可用主机地址
		ip2 := make(net.IP, 4)
		copy(ip2, network)
		increaseIP(ip2)
		return []string{network.String(), ip2.String()}, nil
	}

	// 4. 常规网段 (>= /30)
	var ips []string
	start := network
	end := broadcast

	// 如果跳过网络/广播地址，调整起止范围
	if skipNetworkBroadcast {
		start = make(net.IP, 4)
		copy(start, network)
		increaseIP(start) // 跳过网络地址

		end = make(net.IP, 4)
		copy(end, broadcast)
		decreaseIP(end) // 跳过广播地址
	}

	// 遍历生成 IP
	current := make(net.IP, 4)
	copy(current, start)
	for {
		ips = append(ips, current.String())
		if current.Equal(end) {
			break
		}
		increaseIP(current)
	}

	return ips, nil
}

// incIP IP 地址自增 +1 (IPv4)
func increaseIP(ip net.IP) {
	for i := 3; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// decIP IP 地址自减 -1 (IPv4)
func decreaseIP(ip net.IP) {
	for i := 3; i >= 0; i-- {
		if ip[i] > 0 {
			ip[i]--
			break
		}
		ip[i] = 0xFF
	}
}

// GetNetPreferredIP 获取指定网卡上“最优”的一个IP地址
//   - name: 网卡名称，例如 "eth0"
//   - prefer: IP偏好，取值 "ipv4" 或 "ipv6"，不传默认自动选第一个
func GetNetPreferredIP(name string, prefer ...string) (net.IP, error) {
	ips, err := getValidIPs(name)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, wrapError(ErrNoValidIP, "interface %q", name)
	}

	// 如果指定了偏好，先过滤
	var filtered []net.IP
	per := gokit.GetValueWithDefault("", prefer...)
	switch per {
	case "":
		filtered = ips
	case "ipv4", "ipv6":
		for _, ip := range ips {
			if per == "ipv4" && ip.To4() != nil {
				filtered = append(filtered, ip)
			} else if per == "ipv6" && ip.To4() == nil {
				filtered = append(filtered, ip)
			}
		}
	default:
		return nil, wrapError(ErrInvalidPreference, "%q, want ipv4 or ipv6", per)
	}

	if len(filtered) == 0 {
		return nil, wrapError(ErrNoValidIP, "interface %q with preference %q", name, per)
	}

	// 按作用域排序（Global > ULA > 其他）
	sort.Slice(filtered, func(i, j int) bool {
		return scopePriority(filtered[i]) < scopePriority(filtered[j])
	})

	return filtered[0], nil
}

// getValidIPs 返回指定网卡上所有“可用”的 IP（排除回环、链路本地、未指定）
func getValidIPs(ifaceName string) ([]net.IP, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, wrapError(ErrNoValidIP, "interface %q: %w", ifaceName, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, wrapError(ErrNoValidIP, "interface %q: %w", ifaceName, err)
	}

	var valid []net.IP
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP

		// 排除回环、链路本地、未指定
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			continue
		}

		valid = append(valid, ip)
	}

	return valid, nil
}

func scopePriority(ip net.IP) int {
	if ip.IsGlobalUnicast() && !isULA(ip) {
		// 公网或私有 IPv4，以及非 ULA 的 IPv6 全局单播（如 GUA）
		return 0
	}

	if isULA(ip) {
		// IPv6 ULA，优先级略低于公网/私有 IPv4
		return 1
	}

	// 其他（如链路本地、多播等）
	return 2
}

// isULA 判断是否为 IPv6 唯一本地地址 (fc00::/7)
func isULA(ip net.IP) bool {
	if ip.To4() != nil {
		return false // IPv4 没有 ULA 概念
	}

	// 检查前两个字节：fc00::/7 即 1111110x xxxxxxxx
	if len(ip) >= 2 {
		first := ip[0]
		return first&0xfe == 0xfc // 0xfc 对应 11111100，&0xfe 忽略最低位
	}

	return false
}
