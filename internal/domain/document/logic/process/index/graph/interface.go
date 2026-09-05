package graph

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/extract"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// Extractor 图谱抽取器
type Extractor interface {
	Extract(ctx context.Context, chunk *entity.DocumentChunk) (*extract.RawGraph, error)
}
