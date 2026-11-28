package main

import (
	"flag"
	"strings"
)

// demoConfig Demo 配置
type demoConfig struct {
	id        string
	addr      string
	seedAddrs []string
	maxPeers  int
}

// parseFlags 解析命令行参数
func parseFlags() *demoConfig {
	cfg := &demoConfig{}

	flag.StringVar(&cfg.id, "id", "", "节点唯一标识 (默认自动生成)")
	flag.StringVar(&cfg.addr, "addr", ":9000", "监听端口 (如 :9000)")
	seeds := flag.String("seeds", "", "种子节点地址，逗号分隔 (如 :9000,:9001)")
	flag.IntVar(&cfg.maxPeers, "max-peers", 10, "最大连接数")
	flag.Parse()

	// 解析种子节点
	if *seeds != "" {
		for _, s := range strings.Split(*seeds, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				// 自动补全 localhost
				if strings.HasPrefix(s, ":") {
					s = "127.0.0.1" + s
				}
				cfg.seedAddrs = append(cfg.seedAddrs, s)
			}
		}
	}

	// ID 默认值
	if cfg.id == "" {
		cfg.id = "demo-" + strings.TrimPrefix(cfg.addr, ":")
	}

	return cfg
}
