// Package config 承载 chat 微服务的业务配置。
// Config 为根配置（共享基础配置 + ChatConf 业务配置），各业务域子配置按内聚原则分组。
package config

import (
	sharedcfg "github.com/swiftbit/know-agent/pkg/config"
)

// Config chat 微服务配置：共享基础配置 + 本服务业务配置
type Config struct {
	sharedcfg.BaseConfig
	Chat ChatConf
}

// MustLoad 从 -f 指定的 YAML 加载 chat 配置（默认 etc/chat-dev.yaml）。
func MustLoad() *Config {
	var c Config
	sharedcfg.LoadConfig(&c, "etc/chat-dev.yaml")
	return &c
}

// ChatConf 对话业务总配置
type ChatConf struct {
	Memory         MemoryConf         // 记忆配置
	Rewrite        RewriteConf        // 问题改写配置
	Recommendation RecommendationConf // 推荐配置
	Rag            RagConf            // RAG配置
	SemanticCache  SemanticCacheConf  // 语义缓存配置
	Agent          AgentConf          // Agent配置
}
