package config

// SemanticCacheConf 语义缓存配置
type SemanticCacheConf struct {
	Enabled             bool    `json:",default=false"`                           // 是否启用语义缓存
	Collection          string  `json:",default=semantic_cache"`                  // Milvus 向量集合名
	SimilarityThreshold float32 `json:",default=0.70"`                            // 向量召回粗筛阈值：得分低于该值直接未命中
	DirectHitThreshold  float32 `json:",default=0.90"`                            // 直接命中阈值：最高候选得分达到该值跳过 LLM 判定
	RecallTopK          int     `json:",default=3"`                               // 向量召回候选数（进入灰色区间的候选池大小）
	ConfidenceFloor     float32 `json:",default=0.6"`                             // LLM 判定置信度兜底地板：低于该值强制未命中
	JudgeEnabled        bool    `json:",default=false"`                           // 灰色区间是否启用大模型等价终判（false 时灰色区间保守未命中）
	JudgeTemperature    float32 `json:",default=0.1"`                             // 大模型判定温度
	ReuseAnswer         bool    `json:",default=false"`                           // true=命中时直接复用答案
	WriteTopic          string  `json:",default=know-agent-semantic-cache-write"` // MQ 写入主题（语义缓存启用后写入链路强制异步，消费端按该主题订阅；实际 topic 以部署环境为准）
}
