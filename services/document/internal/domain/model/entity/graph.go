package entity

// 图模型对齐
//   (:Entity)      -[受控关系类型]-> (:Entity)      语义关系（证据内联为数组属性）
//   (:Entity)      -[:IN_COMMUNITY]-> (:Community)  社区归属
// 血缘按 (document_id, task_id) 冻结，落库采用全删全插实现幂等。

// GraphNode 图谱实体节点（对应 Neo4j (:Entity) 节点）
type GraphNode struct {
	EntityKey      string   `json:"entityKey"`      // 实体唯一键 = hash(normalized_name + entity_type)
	Name           string   `json:"name"`           // 实体展示名
	NormalizedName string   `json:"normalizedName"` // 规范化名（检索键，种子/别名匹配目标）
	EntityType     string   `json:"entityType"`     // 实体类型（受控白名单）
	Description    string   `json:"description"`    // 实体描述
	Aliases        []string `json:"aliases"`        // 别名集合
	MentionCount   int      `json:"mentionCount"`   // 提及次数（grounding 锚点数）
	PageRank       float64  `json:"pagerank"`       // PageRank 得分（算法写回）
	RankBoost      float64  `json:"rankBoost"`      // 排序加权分（算法写回）
	Degree         int      `json:"degree"`         // 无向图节点度数（算法写回）
}

// GraphEdge 图谱语义关系（对应 Neo4j 动态受控关系类型边）
// 事实权威与标签权威分离：PredicateQuote 是唯一事实来源，RelationType 仅供检索加权
type GraphEdge struct {
	SourceKey      string   `json:"sourceKey"`      // 源实体 entity_key
	TargetKey      string   `json:"targetKey"`      // 目标实体 entity_key
	RelationType   string   `json:"relationType"`   // 受控关系类型（白名单内或 RELATED_TO 兜底）
	PredicateQuote string   `json:"predicateQuote"` // 原文谓词引用（事实权威）
	Description    string   `json:"description"`    // 关系描述
	Weight         float64  `json:"weight"`         // 合并权重（同 (source,target,type) 累加）
	RankBoost      float64  `json:"rankBoost"`      // 排序加权分（算法写回）
	ChunkIds       []int64  `json:"chunkIds"`       // 证据锚点 chunk id 列表（grounding）
	QuoteTexts     []string `json:"quoteTexts"`     // 原文引用片段（与 ChunkIds 一一对应）
}

// GraphCommunity 社区节点（对应 Neo4j (:Community) 节点，抽取式报告零 LLM 成本）
type GraphCommunity struct {
	CommunityNo int      `json:"communityNo"` // 社区编号（(document_id,task_id) 内唯一，落库 node key 之一）
	Level       int      `json:"level"`       // 层级（固定 0）
	Title       string   `json:"title"`       // 社区标题（top 实体名拼接）
	Summary     string   `json:"summary"`     // 社区摘要
	RankBoost   float64  `json:"rankBoost"`   // 社区排序加权分
	Modularity  float64  `json:"modularity"`  // 模块度（可观测质量指标）
	MemberCount int      `json:"memberCount"` // 成员实体数
	MemberKeys  []string `json:"memberKeys"`  // 成员实体 entity_key 列表（用于 :IN_COMMUNITY 落库）
}

// GraphSnapshot 单篇文档的图谱快照（Neo4j 写事务载体）
type GraphSnapshot struct {
	DocumentId      int64             `json:"documentId"`      // 文档 ID
	TaskId          int64             `json:"taskId"`          // 任务 ID
	KnowledgeBaseId int64             `json:"knowledgeBaseId"` // 知识库 ID
	StrategyPlanId  int64             `json:"strategyPlanId"`  // 策略方案 ID
	Nodes           []*GraphNode      `json:"nodes"`           // 实体节点列表
	Edges           []*GraphEdge      `json:"edges"`           // 语义关系列表
	Communities     []*GraphCommunity `json:"communities"`     // 社区节点列表
}
