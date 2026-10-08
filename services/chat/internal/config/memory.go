package config

// MemoryConf 记忆配置
type MemoryConf struct {
	RecentTranscriptMaxChars int                `json:",default=1024"` // 最近对话记录最大字符数
	QuestionHistoryMaxChars  int                `json:",default=512"`  // 问题历史最大字符数
	HistorySummary           HistorySummaryConf // 历史摘要配置
}

// HistorySummaryConf 历史摘要配置
type HistorySummaryConf struct {
	Enabled                  bool `json:",default=true"` // 是否启用摘要压缩
	HistoryTurns             int  `json:",default=6"`    // 历史轮次
	KeepRecentTurns          int  `json:",default=3"`    // 保留最近轮次
	CompressionBatchTurns    int  `json:",default=3"`    // 压缩批次轮次
	MaxChars                 int  `json:",default=1024"` // 摘要最大字符数
	RecentTranscriptMaxChars int  `json:",default=2200"` // 最近对话记录最大字符数
}
