package config

import "time"

// RagConf RAG配置
type RagConf struct {
	ChannelTimeout     time.Duration `json:",default=5s"`  // 通道超时时间
	SubQuestionTimeout time.Duration `json:",default=12s"` // 子问题超时时间
	CandidateTopK      int           `json:",default=40"`  // 候选集TopK
	FinalTopK          int           `json:",default=6"`   // 最终结果TopK
	Keyword            *KeywordConf  // 关键词通道
	Vector             *VectorConf   // 向量通道
	Graph              *GraphRagConf // 图RAG通道
	Rerank             *RerankConf   // 重排序通道
	Hybrid             *HybridConf   // 混合检索权重
	Scope              *ScopeConf    // 检索范围收窄阈值
}

// ScopeConf 检索范围收窄阈值配置
type ScopeConf struct {
	MinIntentScore      float64 `json:",default=0.4"`  // 节点计入命中的次级阈值
	ConfidenceThreshold float64 `json:",default=0.6"`  // topScore 达到则判定为 DIRECTED
	SupplementRatio     float64 `json:",default=0.25"` // 补充路比例，区间 [0,1]
}

// KeywordConf 关键词通道配置
type KeywordConf struct {
	Enabled            bool    `json:",default=true"` // 是否启用关键词
	TopK               int     `json:",default=10"`   // 关键词检索TopK
	RelativeScoreFloor float64 `json:",default=0.35"` // 关键词相对分数阈值
}

// VectorConf 向量通道配置
type VectorConf struct {
	TopK          int     `json:",default=10"`   // 向量检索TopK
	MinSimilarity float64 `json:",default=0.45"` // 相似度阈值
}

// GraphRagConf 图RAG配置
type GraphRagConf struct {
	Enabled       bool `json:",default=true"` // 启用开关
	SearchMaxHops int  `json:",default=2"`    // 检索最大跳数（硬限 2）
	SearchTopK    int  `json:",default=10"`   // 检索返回上限
}

// HybridConf 混合检索权重
type HybridConf struct {
	VectorWeight        float64 `json:",default=1.0"`
	KeywordWeight       float64 `json:",default=1.0"`
	GraphRagWeight      float64 `json:",default=1.1"`
	RankWeight          float64 `json:",default=1.0"`
	OriginalScoreWeight float64 `json:",default=0.08"`
	MetadataBoostWeight float64 `json:",default=0.04"`
	MaxMetadataBoost    float64 `json:",default=1.0"`
}

// RerankConf 重排序通道配置：knowledge 仅消费启停与 TopN，重排序调用的地址/密钥/超时等
type RerankConf struct {
	Enabled bool `json:",default=true"` // 是否启用重排序
	TopN    int  `json:",default=24"`   // 重排序TopN
}
