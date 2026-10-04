// Package grpcclient 承载 document 服务出站调用其他服务的 gRPC 客户端封装。
package grpcclient

import (
	"context"

	"google.golang.org/grpc"

	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/proto/knowagent/knowledge/v1"
	den "github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	dvo "github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// KnowledgeClient 实现 document 域出站消费 knowledge 服务的接口：
//   - adapter.KnowledgeGateway（RequireEnabled）
//   - GraphTypeProvider（GraphTypes）
//   - IndexingConfigResolver（Resolve）
type KnowledgeClient struct {
	client knowledgev1.KnowledgeServiceClient
}

func NewKnowledgeClient(cc *grpc.ClientConn) *KnowledgeClient {
	return &KnowledgeClient{client: knowledgev1.NewKnowledgeServiceClient(cc)}
}

func (c *KnowledgeClient) RequireEnabled(ctx context.Context, knowledgeBaseId int64) (*den.KnowledgeBase, error) {
	resp, err := c.client.RequireEnabled(ctx, &knowledgev1.RequireEnabledRequest{KnowledgeBaseId: knowledgeBaseId})
	if err != nil {
		return nil, err
	}
	kb := resp.GetKnowledgeBase()
	return &den.KnowledgeBase{ID: kb.GetId(), BaseName: kb.GetBaseName()}, nil
}

func (c *KnowledgeClient) GraphTypes(ctx context.Context, knowledgeBaseId int64) ([]string, []string, error) {
	resp, err := c.client.GraphTypes(ctx, &knowledgev1.GraphTypesRequest{KnowledgeBaseId: knowledgeBaseId})
	if err != nil {
		return nil, nil, err
	}
	return resp.GetEntityTypes(), resp.GetRelationTypes(), nil
}

func (c *KnowledgeClient) Resolve(ctx context.Context, document *den.Document) *dvo.IndexingOptions {
	resp, err := c.client.Resolve(ctx, &knowledgev1.ResolveRequest{KnowledgeBaseId: document.KnowledgeBaseId})
	if err != nil {
		return &dvo.IndexingOptions{}
	}
	return protoIndexingOptionsToVO(resp)
}

func protoIndexingOptionsToVO(p *knowledgev1.IndexingOptions) *dvo.IndexingOptions {
	if p == nil {
		return nil
	}
	chunk := p.GetChunk()
	graph := p.GetGraphRag()
	raptor := p.GetRaptor()
	return &dvo.IndexingOptions{
		Chunk: dvo.ChunkOptions{
			ChildRecursiveMaxChars:           int(chunk.GetChildRecursiveMaxChars()),
			ChildRecursiveOverlapChars:       int(chunk.GetChildRecursiveOverlapChars()),
			ChildSemanticMaxChars:            int(chunk.GetChildSemanticMaxChars()),
			ChildSemanticMinChars:            int(chunk.GetChildSemanticMinChars()),
			ChildSemanticSimilarityThreshold: chunk.GetChildSemanticSimilarityThreshold(),
			ParentChunkMaxChars:              int(chunk.GetParentChunkMaxChars()),
			ParentChunkOverlapChars:          int(chunk.GetParentChunkOverlapChars()),
			ParentSemanticMaxChars:           int(chunk.GetParentSemanticMaxChars()),
			ParentSemanticMinChars:           int(chunk.GetParentSemanticMinChars()),
		},
		GraphRag: dvo.GraphRagBuildOptions{
			GraphRagBuildEnabled: graph.GetGraphRagBuildEnabled(),
		},
		Raptor: dvo.RaptorBuildOptions{
			RaptorBuildEnabled:        raptor.GetRaptorBuildEnabled(),
			RaptorMaxClusterSize:      int(raptor.GetRaptorMaxClusterSize()),
			RaptorMaxLevels:           int(raptor.GetRaptorMaxLevels()),
			RaptorLlmSummaryEnabled:   raptor.GetRaptorLlmSummaryEnabled(),
			RaptorSummaryQualityFloor: raptor.GetRaptorSummaryQualityFloor(),
		},
	}
}
