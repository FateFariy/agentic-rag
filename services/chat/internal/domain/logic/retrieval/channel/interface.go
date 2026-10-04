package channel

import (
	"context"

	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/vo"
)

// KeywordRetriever 关键词检索器
type KeywordRetriever interface {
	// SearchByKeyword  按关键词检索
	SearchByKeyword(ctx context.Context, query *Input) ([]*vo.DocumentChunk, error)
}

// VectorRetriever 向量检索器
type VectorRetriever interface {
	// SearchByVector 按向量检索
	SearchByVector(ctx context.Context, query *Input) ([]*vo.DocumentChunk, error)
}

// GraphRetriever 图谱检索器
type GraphRetriever interface {
	// SearchBySeed 种子实体命中 + n-hop 变长路径扩展 + 证据回流
	SearchBySeed(ctx context.Context, input *Input) ([]*vo.DocumentChunk, error)
}
