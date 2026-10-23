package grpc

import (
	"context"

	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/aggregate"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

// 编译期断言：KnowledgeGrpcServer 实现 KnowledgeServiceServer
var _ knowledgev1.KnowledgeServiceServer = (*KnowledgeGrpcServer)(nil)

// KnowledgeGrpcServer knowledge 域 gRPC 服务端，协议转换与 client/knowledge.go 严格镜像对称
type KnowledgeGrpcServer struct {
	knowledgev1.UnimplementedKnowledgeServiceServer
	retrieval logic.KnowledgeBaseRetrievalLogic
	kbl       logic.KnowledgeBaseLogic
}

// NewKnowledgeGrpcServer 创建 knowledge gRPC 服务端
func NewKnowledgeGrpcServer(retrieval logic.KnowledgeBaseRetrievalLogic, kbl logic.KnowledgeBaseLogic) *KnowledgeGrpcServer {
	return &KnowledgeGrpcServer{retrieval: retrieval, kbl: kbl}
}

func (s *KnowledgeGrpcServer) DetermineKnowledgeScope(ctx context.Context, req *knowledgev1.DetermineKnowledgeScopeRequest) (*knowledgev1.KnowledgeBaseSelectionSnapshot, error) {
	snapshot, err := s.retrieval.DetermineKnowledgeScope(ctx, req.GetSelectMode(), req.GetKbIds(), req.GetQuestion())
	if err != nil {
		return nil, err
	}
	return snapshotToProto(snapshot), nil
}

func (s *KnowledgeGrpcServer) Resolve(ctx context.Context, req *knowledgev1.ResolveRequest) (*knowledgev1.IndexingOptions, error) {
	options := s.kbl.ResolveIndexingOptions(ctx, req.GetKnowledgeBaseId())
	protoOptions := indexingOptionsToProto(options)
	if protoOptions == nil {
		protoOptions = &knowledgev1.IndexingOptions{}
	}
	return protoOptions, nil
}

func (s *KnowledgeGrpcServer) RequireEnabled(ctx context.Context, req *knowledgev1.RequireEnabledRequest) (*knowledgev1.RequireEnabledResponse, error) {
	base, err := s.kbl.GetEnabledKnowledgeBase(ctx, req.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	return &knowledgev1.RequireEnabledResponse{KnowledgeBase: knowledgeBaseToProto(base)}, nil
}

func (s *KnowledgeGrpcServer) GraphTypes(ctx context.Context, req *knowledgev1.GraphTypesRequest) (*knowledgev1.GraphTypesResponse, error) {
	entityTypes, relationTypes, err := s.kbl.GraphTypes(ctx, req.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	return &knowledgev1.GraphTypesResponse{EntityTypes: entityTypes, RelationTypes: relationTypes}, nil
}

// ==================== 私有转换（与 grpc/client 端映射反向对称） ====================

func snapshotToProto(s *aggregate.KnowledgeBaseSelectionSnapshot) *knowledgev1.KnowledgeBaseSelectionSnapshot {
	if s == nil {
		return nil
	}
	return &knowledgev1.KnowledgeBaseSelectionSnapshot{
		SelectionMode:              s.SelectionMode,
		SelectedKnowledgeBaseIds:   s.SelectedKnowledgeBaseIDs(),
		SelectedKnowledgeBaseNames: s.SelectedKnowledgeBaseNames(),
		SelectedKnowledgeBases:     knowledgeBaseListToProto(s.SelectedKnowledgeBases),
		AllowedDocuments:           documentMetadataListToProto(s.AllowedDocuments),
		RagRuntimeOptions:          ragRuntimeOptionsToProto(s.RagRuntimeOptions),
		NarrowedDocumentIds:        s.NarrowedDocumentIds,
	}
}

func knowledgeBaseListToProto(items []*entity.KnowledgeBase) []*knowledgev1.KnowledgeBase {
	return utils.Map(items, func(item *entity.KnowledgeBase) *knowledgev1.KnowledgeBase {
		return knowledgeBaseToProto(item)
	})
}

// knowledgeBaseToProto 转换完整知识库（跨服务传递全部配置字段）
func knowledgeBaseToProto(item *entity.KnowledgeBase) *knowledgev1.KnowledgeBase {
	if item == nil {
		return nil
	}
	var isDefault int32
	if item.IsDefault != nil {
		isDefault = int32(*item.IsDefault)
	}
	return &knowledgev1.KnowledgeBase{
		Id:                  item.ID,
		BaseName:            item.BaseName,
		Description:         item.Description,
		RetrievalConfigJson: string(item.RetrievalConfigJson),
		GraphRagConfigJson:  string(item.GraphRagConfigJson),
		MetadataFilterJson:  string(item.MetadataFilterJson),
		IsDefault:           isDefault,
		SortOrder:           int32(item.SortOrder),
	}
}

func documentMetadataListToProto(items []*vo.DocumentMetadata) []*documentv1.DocumentMetadata {
	out := make([]*documentv1.DocumentMetadata, 0, len(items))
	for _, item := range items {
		if item != nil {
			out = append(out, &documentv1.DocumentMetadata{
				DocumentId:        item.DocumentId,
				DocumentName:      item.DocumentName,
				KnowledgeBaseId:   item.KnowledgeBaseId,
				KnowledgeBaseName: item.KnowledgeBaseName,
				LastIndexTaskId:   item.LastIndexTaskId,
			})
		}
	}
	return out
}

func ragRuntimeOptionsToProto(o *vo.RagRuntimeOptions) *knowledgev1.RagRuntimeOptions {
	if o == nil {
		return nil
	}
	return &knowledgev1.RagRuntimeOptions{
		VectorTopK:                int32(o.VectorTopK),
		KeywordTopK:               int32(o.KeywordTopK),
		GraphRagTopK:              int32(o.GraphRagTopK),
		GraphRagMaxHops:           int32(o.GraphRagMaxHops),
		CandidateTopK:             int32(o.CandidateTopK),
		RerankCandidateTopK:       int32(o.RerankCandidateTopK),
		FinalTopK:                 int32(o.FinalTopK),
		RerankEnabled:             o.RerankEnabled,
		ChannelTimeoutMs:          o.ChannelTimeout.Milliseconds(),
		SubQuestionTimeoutMs:      o.SubQuestionTimeout.Milliseconds(),
		MinVectorSimilarity:       o.MinVectorSimilarity,
		KeywordRelativeScoreFloor: o.KeywordRelativeScoreFloor,
		KeywordChannelEnabled:     o.KeywordChannelEnabled,
		GraphRagChannelEnabled:    o.GraphRagChannelEnabled,
		Hybrid:                    hybridOptionsToProto(o.Hybrid),
	}
}

func hybridOptionsToProto(o *vo.HybridOptions) *knowledgev1.HybridOptions {
	if o == nil {
		return nil
	}
	return &knowledgev1.HybridOptions{
		VectorWeight:        o.VectorWeight,
		KeywordWeight:       o.KeywordWeight,
		GraphRagWeight:      o.GraphRagWeight,
		RankWeight:          o.RankWeight,
		OriginalScoreWeight: o.OriginalScoreWeight,
		MetadataBoostWeight: o.MetadataBoostWeight,
		MaxMetadataBoost:    o.MaxMetadataBoost,
	}
}

func indexingOptionsToProto(o *vo.IndexingOptions) *knowledgev1.IndexingOptions {
	if o == nil {
		return nil
	}
	return &knowledgev1.IndexingOptions{
		Chunk: &knowledgev1.ChunkOptions{
			ChildRecursiveMaxChars:           int32(o.Chunk.ChildRecursiveMaxChars),
			ChildRecursiveOverlapChars:       int32(o.Chunk.ChildRecursiveOverlapChars),
			ChildSemanticMaxChars:            int32(o.Chunk.ChildSemanticMaxChars),
			ChildSemanticMinChars:            int32(o.Chunk.ChildSemanticMinChars),
			ChildSemanticSimilarityThreshold: o.Chunk.ChildSemanticSimilarityThreshold,
			ParentChunkMaxChars:              int32(o.Chunk.ParentChunkMaxChars),
			ParentChunkOverlapChars:          int32(o.Chunk.ParentChunkOverlapChars),
			ParentSemanticMaxChars:           int32(o.Chunk.ParentSemanticMaxChars),
			ParentSemanticMinChars:           int32(o.Chunk.ParentSemanticMinChars),
		},
		GraphRag: &knowledgev1.GraphRagBuildOptions{
			GraphRagBuildEnabled: o.GraphRag.GraphRagBuildEnabled,
		},
	}
}
