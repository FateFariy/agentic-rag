package channel

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// GraphRAGRetrievalChannel 图谱检索通道（M1：种子实体 + n-hop 变长路径 + 社区召回 + 证据回流）
type GraphRAGRetrievalChannel struct {
	retriever GraphRetriever
}

// NewGraphRetrievalChannel 创建图谱检索通道
func NewGraphRetrievalChannel(retriever GraphRetriever) *GraphRAGRetrievalChannel {
	return &GraphRAGRetrievalChannel{retriever: retriever}
}

// Name 通道名称
func (c *GraphRAGRetrievalChannel) Name() string {
	return enum.RetrievalChannelGraphRAG
}

// Retrieve 执行图谱检索，与 vector 通道同构：
// NewInput 构建请求 → Validate → SearchBySeed（证据回流 []*vo.DocumentChunk）→ MinimumScore 过滤。
// 图谱为增强依赖：未装配 / 图意图未请求 / 通道未计划 / 查询失败时返回空结果，不阻断其余通道。
func (c *GraphRAGRetrievalChannel) Retrieve(ctx context.Context, input *retrieval.ExecutionInput) (*retrieval.RetrievalChannelResult, error) {
	if c.retriever == nil || input == nil {
		return nil, nil
	}
	if input.GraphIntent == nil || !input.GraphIntent.Requested {
		return nil, nil
	}
	chanInput, err := NewInput(c.Name(), input)
	if err != nil {
		return nil, err
	}
	if !chanInput.Validate() {
		return nil, fmt.Errorf("invalid value")
	}
	docs, err := c.retriever.SearchBySeed(ctx, chanInput)
	if err != nil {
		logx.Errorf("图谱检索失败: question='%s', error=%v", chanInput.Question, err)
		return nil, err
	}
	channel, _ := input.RequireChannel(c.Name())
	accepted := utils.Filter(docs, func(doc *vo.DocumentChunk) bool {
		return doc != nil && doc.Score >= channel.MinimumScore
	})
	return &retrieval.RetrievalChannelResult{
		Name:              c.Name(),
		RawDocuments:      docs,
		AcceptedDocuments: accepted,
	}, nil
}
