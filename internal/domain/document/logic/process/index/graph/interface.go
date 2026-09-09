package graph

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/extract"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// Extractor 图谱抽取器
type Extractor interface {
	// Extract 抽取单个块；entityTypes/relationTypes 为该知识库允许的实体/关系类型白名单，
	// 为空时由实现回退到全局默认白名单。
	Extract(ctx context.Context, chunk *entity.DocumentChunk, entityTypes, relationTypes []string) (*extract.RawGraph, error)
}

// CommunityReporter 社区摘要增强（M3：LLM 社区报告）。
// 社区报告为效果增强依赖：失败时由调用方回退抽取式摘要，不阻断图谱构建。
type CommunityReporter interface {
	// Report 生成单个社区的 LLM 摘要
	Report(ctx context.Context, community *entity.GraphCommunity, members []*entity.GraphNode) (summary string, err error)
}
