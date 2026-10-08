package rpc

import (
	"context"

	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// KnowledgeClient 实现 document 域出站消费 knowledge 服务的接口：
//   - adapter.KnowledgeGateway（RequireEnabled）
//   - GraphTypeProvider（GraphTypes）
//   - IndexingConfigResolver（Resolve）
type KnowledgeClient struct {
	client knowledgev1.KnowledgeServiceClient
}

func NewKnowledgeClient(client knowledgev1.KnowledgeServiceClient) *KnowledgeClient {
	return &KnowledgeClient{client: client}
}

func (c *KnowledgeClient) RequireEnabled(ctx context.Context, knowledgeBaseId int64) (*entity.KnowledgeBase, error) {
	resp, err := c.client.RequireEnabled(ctx, &knowledgev1.RequireEnabledRequest{KnowledgeBaseId: knowledgeBaseId})
	if err != nil {
		return nil, err
	}
	kb := resp.GetKnowledgeBase()
	return &entity.KnowledgeBase{ID: kb.GetId(), BaseName: kb.GetBaseName()}, nil
}

func (c *KnowledgeClient) GraphTypes(ctx context.Context, knowledgeBaseId int64) ([]string, []string, error) {
	resp, err := c.client.GraphTypes(ctx, &knowledgev1.GraphTypesRequest{KnowledgeBaseId: knowledgeBaseId})
	if err != nil {
		return nil, nil, err
	}
	return resp.GetEntityTypes(), resp.GetRelationTypes(), nil
}

func (c *KnowledgeClient) Resolve(ctx context.Context, document *entity.Document) *vo.IndexingOptions {
	resp, err := c.client.Resolve(ctx, &knowledgev1.ResolveRequest{KnowledgeBaseId: document.KnowledgeBaseId})
	if err != nil {
		return &vo.IndexingOptions{}
	}
	return protoIndexingOptionsToVO(resp)
}

func protoIndexingOptionsToVO(p *knowledgev1.IndexingOptions) *vo.IndexingOptions {
	if p == nil {
		return nil
	}
	chunk := p.GetChunk()
	graph := p.GetGraphRag()
	return &vo.IndexingOptions{
		Chunk: vo.ChunkOptions{
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
		GraphRag: vo.GraphRagBuildOptions{
			GraphRagBuildEnabled: graph.GetGraphRagBuildEnabled(),
		},
	}
}
