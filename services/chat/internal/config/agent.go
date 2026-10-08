package config

import "time"

// AgentConf Agent配置
type AgentConf struct {
	RecommendationEnabled  bool          `json:",default=true"` // 是否启用推荐追问
	MaxModelCallsPerRun    int           `json:",default=8"`    // 每次最大模型调用次数
	MaxModelCallsPerThread int           `json:",default=40"`   // 每次最大模型调用线程数
	MaxToolCallsPerRun     int           `json:",default=6"`    // 每次最大工具调用次数
	MaxToolCallsPerThread  int           `json:",default=30"`   // 每次最大工具调用线程数
	HistoryPreviewTurns    int           `json:",default=4"`    // 预览历史轮次
	RecommendationTimeout  time.Duration `json:",default=5s"`   // 推荐生成超时时间
	SystemPrompt           string        `json:",optional"`     // 系统提示
}

// RecommendationConf 推荐追问配置
type RecommendationConf struct {
	Enabled             bool          `json:",default=true"` // 是否启用推荐追问
	Timeout             time.Duration `json:",default=5s"`   // 推荐生成超时时间
	HistoryPreviewTurns int           `json:",default=3"`    // 预览历史轮次
}
