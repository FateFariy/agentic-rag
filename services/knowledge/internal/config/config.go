// Package config 承载 knowledge 微服务的业务配置。
// knowledge 仅消费 RAG 运行时参数与切分参数，故自带 ChatConf（实际用其 Rag 段）
// 与 ChunkConf 的副本，读取同一份配置文件，实现与 chat/document 的配置解耦。
package config

import (
	sharedcfg "github.com/swiftbit/know-agent/pkg/config"
)

// Config knowledge 微服务配置：共享基础配置 + 本服务业务配置副本
type Config struct {
	sharedcfg.BaseConfig
	Chat  ChatConf
	Chunk ChunkConf
}

// MustLoad 从 -f 指定的 YAML 加载 knowledge 配置（默认 etc/config-dev.yaml）。
func MustLoad() *Config {
	var c Config
	sharedcfg.LoadConfig(&c)
	return &c
}
