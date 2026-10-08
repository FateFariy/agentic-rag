package config

import "time"

// RagConf RAG配置（knowledge 直接消费，不再经 Chat 外壳）
type RagConf struct {
	Enabled                        bool          `json:",default=true"` // 是否启用RAG
	ChannelTimeout                 time.Duration `json:",default=5s"`   // 通道超时时间
	SubQuestionTimeout             time.Duration `json:",default=12s"`  // 子问题超时时间
	CandidateTopK                  int           `json:",default=40"`   // 候选集TopK
	FinalTopK                      int           `json:",default=6"`    // 最终结果TopK
	ParentEvidenceMaxChars         int           `json:",default=2200"` // 父证据最大字符数
	PlanningHistoryMaxChars        int           `json:",default=1600"` // 规划历史最大字符数
	QuestionHistoryMaxChars        int           `json:",default=1000"` // 问题历史最大字符数
	TotalEvidenceMaxChars          int           `json:",default=5200"` // 总证据最大字符数
	PerSubQuestionEvidenceMaxChars int           `json:",default=2200"` // 每个子问题证据最大字符数
	Keyword                        *KeywordConf  // 关键词通道
	Vector                         *VectorConf   // 向量通道
	Graph                          *GraphRagConf // 图RAG通道
	Rerank                         *RerankConf   // 重排序通道
	Hybrid                         *HybridConf   // 混合检索权重
}

// KeywordConf 关键词通道配置
type KeywordConf struct {
	Enabled            bool    `json:",default=true"` // 是否启用关键词
	TopK               int     `json:",default=10"`   // 关键词检索TopK
	RelativeScoreFloor float64 `json:",default=0.35"` // 关键词相对分数阈值
}

// VectorConf 向量通道配置
type VectorConf struct {
	Enabled       bool    `json:",default=true"` // 是否启用向量
	TopK          int     `json:",default=10"`   // 向量检索TopK
	MinSimilarity float64 `json:",default=0.45"` // 相似度阈值
}

// GraphRagConf 图RAG相关配置
type GraphRagConf struct {
	Enabled                     bool    `json:",default=true"` // 启用开关
	ExtractBatch                int     `json:",default=1"`    // 每批 chunk 数（抽取提示）
	ExtractConcurrency          int     `json:",default=4"`    // 抽取并发批次数（1 表示串行）
	ChunkMaxChars               int     `json:",default=1200"` // 单 chunk 抽取截断字符数
	MaxEntities                 int     `json:",default=300"`  // 单文档实体上限（超限按 PageRank 截断）
	MaxRelations                int     `json:",default=500"`  // 单文档关系上限
	MaxCommunities              int     `json:",default=50"`   // 单文档社区上限
	CommunityTopNodes           int     `json:",default=5"`    // 社区标题/摘要取 top 实体数
	CommunityReportTopN         int     `json:",default=12"`   // M3 LLM 社区报告预算闸门：仅前 N 个重要社区生成 LLM 摘要（0 关闭）
	LouvainEnabled              bool    `json:",default=true"` // 是否启用 Louvain 细化（false 时仅并查集连通分量）
	LouvainGamma                float64 `json:",default=1.0"`  // Louvain 分辨率（γ）
	PageRankDamping             float64 `json:",default=0.85"` // PageRank 阻尼系数
	PageRankMaxIter             int     `json:",default=30"`   // PageRank 最大迭代轮数
	SearchMaxHops               int     `json:",default=2"`    // 检索最大跳数（硬限 2）
	SearchTopK                  int     `json:",default=10"`   // 检索返回上限
	TypedChunkProjectionEnabled bool    `json:",default=true"` // M3 Typed chunk 投影：图卡片写入向量库（增强依赖）
	TypedCardTopK               int     `json:",default=100"`  // M3 图卡片总预算（实体:关系:社区 = 2:5:3 分配）
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

// RerankConf 重排序通道配置
type RerankConf struct {
	Enabled        bool          `json:",default=true"` // 是否启用重排序
	URL            string        `json:",optional"`     // 重排序API地址
	ApiKey         string        `json:",optional"`     // 重排序API密钥
	Model          string        `json:",optional"`     // 重排序模型
	TopN           int           `json:",default=24"`   // 重排序TopN
	ConnectTimeout time.Duration `json:",default=3s"`   // 连接超时时间
	ReadTimeout    time.Duration `json:",default=6s"`   // 读取超时时间
	ScoreThreshold float64       `json:",default=0.0"`  // 重排序分数阈值
}
