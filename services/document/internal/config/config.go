package config

import (
	sharedcfg "github.com/swiftbit/know-agent/pkg/config"
)

// Config document 微服务配置：共享基础配置 + 本服务业务配置
// Rpc 服务端配置复用 BaseConfig.Rpc（可选，未配置时由 main.go 填入默认监听端口）
type Config struct {
	sharedcfg.BaseConfig
	Rag   RagConf
	Chunk ChunkConf
}

// MustLoad 从 -f 指定的 YAML 加载 document 配置（默认 etc/document-dev.yaml）。
func MustLoad() *Config {
	var c Config
	sharedcfg.LoadConfig(&c, "etc/document-dev.yaml")
	return &c
}
