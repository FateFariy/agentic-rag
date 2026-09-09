package vo

import "github.com/swiftbit/know-agent/internal/domain/document/model/enum"

// GraphRagBuildResult GraphRAG 构建结果
type GraphRagBuildResult struct {
	EntityCount             int                          `json:"entityCount"`             // 实体数量
	RelationCount           int                          `json:"relationCount"`           // 关系数量
	EvidenceCount           int                          `json:"evidenceCount"`           // 证据锚点数
	CommunityCount          int                          `json:"communityCount"`          // 社区数量
	GraphPersistenceOutcome enum.GraphPersistenceOutcome `json:"graphPersistenceOutcome"` // 图谱持久化结果
	GraphPersistenceReason  string                       `json:"graphPersistenceReason"`  // 结果说明/降级原因
	KgCommitted             bool                         `json:"kgCommitted"`             // 图谱是否已提交
}

// RaptorBuildResult RAPTOR 构建结果
type RaptorBuildResult struct {
	NodeCount           int    `json:"nodeCount"`
	LevelCount          int    `json:"levelCount"`
	SourceChunkCount    int    `json:"sourceChunkCount"`
	SourceQualityReport string `json:"sourceQualityReport"`
	SavedQualityReport  string `json:"savedQualityReport"`
}
