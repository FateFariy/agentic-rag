package server

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/proto/knowagent/document/v1"
	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/proto/knowagent/knowledge/v1"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/aggregate"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
)

// 编译期断言：KnowledgeServer 实现 KnowledgeServiceServer。
var _ knowledgev1.KnowledgeServiceServer = (*KnowledgeServer)(nil)

// KnowledgeServer knowledge 域 gRPC 服务端，协议转换与 client/knowledge.go 严格镜像对称。
// 仅依赖 knowledge 自有领域类型与 proto 契约，不引用 chat/document 域类型。
type KnowledgeServer struct {
	knowledgev1.UnimplementedKnowledgeServiceServer
	adapter *gateway.KnowledgeAdapter
}

// NewKnowledgeServer 创建 knowledge gRPC 服务端。
func NewKnowledgeServer(adapter *gateway.KnowledgeAdapter) *KnowledgeServer {
	return &KnowledgeServer{adapter: adapter}
}

func (s *KnowledgeServer) DetermineKnowledgeScope(ctx context.Context, req *knowledgev1.DetermineKnowledgeScopeRequest) (*knowledgev1.KnowledgeBaseSelectionSnapshot, error) {
	snapshot, err := s.adapter.DetermineKnowledgeScope(ctx, req.GetSelectMode(), req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return snapshotToProto(snapshot), nil
}

func (s *KnowledgeServer) Route(ctx context.Context, req *knowledgev1.RouteRequest) (*knowledgev1.KnowledgeRouteDecision, error) {
	decision, err := s.adapter.Route(ctx, routeContextFromProto(req.GetInput()))
	if err != nil {
		return nil, err
	}
	return decisionToProto(decision), nil
}

func (s *KnowledgeServer) RecordShadowRoute(ctx context.Context, req *knowledgev1.RecordShadowRouteRequest) (*emptypb.Empty, error) {
	if err := s.adapter.RecordShadowRoute(ctx, routeContextFromProto(req.GetInput())); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *KnowledgeServer) Resolve(ctx context.Context, req *knowledgev1.ResolveRequest) (*knowledgev1.IndexingOptions, error) {
	options := s.adapter.Resolve(ctx, req.GetKnowledgeBaseId())
	protoOptions := indexingOptionsToProto(options)
	if protoOptions == nil {
		protoOptions = &knowledgev1.IndexingOptions{}
	}
	return protoOptions, nil
}

func (s *KnowledgeServer) RequireEnabled(ctx context.Context, req *knowledgev1.RequireEnabledRequest) (*knowledgev1.RequireEnabledResponse, error) {
	base, err := s.adapter.RequireEnabled(ctx, req.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	return &knowledgev1.RequireEnabledResponse{KnowledgeBase: knowledgeBaseToProto(base)}, nil
}

func (s *KnowledgeServer) GraphTypes(ctx context.Context, req *knowledgev1.GraphTypesRequest) (*knowledgev1.GraphTypesResponse, error) {
	entityTypes, relationTypes, err := s.adapter.GraphTypes(ctx, req.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	return &knowledgev1.GraphTypesResponse{EntityTypes: entityTypes, RelationTypes: relationTypes}, nil
}

// ==================== 私有转换（与 grpc/client 端映射反向对称） ====================

// routeContextFromProto 将 proto 路由输入转换为 knowledge 自有路由上下文。
func routeContextFromProto(p *knowledgev1.KnowledgeRouteInput) *route.Context {
	if p == nil {
		return nil
	}
	routeCtx := route.NewRouteContext(p.GetQuestion(), p.GetRewriteQuestion(), p.GetSelectedKnowledgeBaseIds(), p.GetAllowedDocumentIds())
	routeCtx.ConversationId = p.GetConversationId()
	routeCtx.ExchangeId = p.GetExchangeId()
	routeCtx.SelectedDocumentId = p.GetSelectedDocumentId()
	routeCtx.KnowledgeBaseSelectionMode = p.GetKnowledgeBaseSelectionMode()
	routeCtx.SelectedKnowledgeBaseNames = p.GetSelectedKnowledgeBaseNames()
	return routeCtx
}

func snapshotToProto(s *aggregate.KnowledgeBaseSelectionSnapshot) *knowledgev1.KnowledgeBaseSelectionSnapshot {
	if s == nil {
		return nil
	}
	return &knowledgev1.KnowledgeBaseSelectionSnapshot{
		SelectionMode:              string(s.SelectionMode),
		SelectedKnowledgeBaseIds:   s.SelectedKnowledgeBaseIds,
		SelectedKnowledgeBaseNames: s.SelectedKnowledgeBaseNames,
		SelectedKnowledgeBases:     knowledgeBaseListToProto(s.SelectedKnowledgeBases),
		AllowedDocuments:           documentMetadataListToProto(s.AllowedDocuments),
		RagRuntimeOptions:          ragRuntimeOptionsToProto(s.RagRuntimeOptions),
	}
}

func knowledgeBaseListToProto(items []*entity.KnowledgeBase) []*knowledgev1.KnowledgeBase {
	out := make([]*knowledgev1.KnowledgeBase, 0, len(items))
	for _, item := range items {
		out = append(out, knowledgeBaseToProto(item))
	}
	return out
}

// knowledgeBaseToProto 转换完整知识库（跨服务传递全部配置字段）。
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
		EmbeddingModel:      item.EmbeddingModel,
		RetrievalConfigJson: string(item.RetrievalConfigJson),
		GraphRagConfigJson:  string(item.GraphRagConfigJson),
		RaptorConfigJson:    string(item.RaptorConfigJson),
		MetadataFilterJson:  string(item.MetadataFilterJson),
		IsDefault:           isDefault,
		SortOrder:           int32(item.SortOrder),
	}
}

func documentMetadataListToProto(items []*vo.DocumentMetadata) []*documentv1.DocumentMetadata {
	out := make([]*documentv1.DocumentMetadata, 0, len(items))
	for _, item := range items {
		if item == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, &documentv1.DocumentMetadata{
			DocumentId:        item.DocumentId,
			DocumentName:      item.DocumentName,
			KnowledgeBaseId:   item.KnowledgeBaseId,
			KnowledgeBaseName: item.KnowledgeBaseName,
			LastIndexTaskId:   item.LastIndexTaskId,
		})
	}
	return out
}

func decisionToProto(d *vo.KnowledgeRouteDecision) *knowledgev1.KnowledgeRouteDecision {
	if d == nil {
		return nil
	}
	return &knowledgev1.KnowledgeRouteDecision{
		Scopes:          scopeCandidatesToProto(d.Scopes),
		Topics:          topicCandidatesToProto(d.Topics),
		Documents:       documentCandidatesToProto(d.Documents),
		Confidence:      d.Confidence,
		RouteStatus:     string(d.RouteStatus),
		Reason:          d.Reason,
		Source:          d.Source,
		Degraded:        d.IsDegraded,
		DegradedReasons: d.DegradedReasons,
	}
}

func scopeCandidatesToProto(items []*vo.ScopeRouteCandidate) []*knowledgev1.ScopeRouteCandidate {
	out := make([]*knowledgev1.ScopeRouteCandidate, 0, len(items))
	for _, item := range items {
		if item == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, &knowledgev1.ScopeRouteCandidate{
			ScopeId:   item.ScopeId,
			ScopeName: item.ScopeName,
			Score:     item.Score,
			Reason:    item.Reason,
			Source:    item.Source,
			Features:  item.Features,
		})
	}
	return out
}

func topicCandidatesToProto(items []*vo.TopicRouteCandidate) []*knowledgev1.TopicRouteCandidate {
	out := make([]*knowledgev1.TopicRouteCandidate, 0, len(items))
	for _, item := range items {
		if item == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, &knowledgev1.TopicRouteCandidate{
			TopicId:   item.TopicId,
			TopicName: item.TopicName,
			ScopeId:   item.ScopeId,
			Score:     item.Score,
			Reason:    item.Reason,
			Source:    item.Source,
			Features:  item.Features,
		})
	}
	return out
}

func documentCandidatesToProto(items []*vo.DocumentRouteCandidate) []*knowledgev1.DocumentRouteCandidate {
	out := make([]*knowledgev1.DocumentRouteCandidate, 0, len(items))
	for _, item := range items {
		if item == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, &knowledgev1.DocumentRouteCandidate{
			DocumentId:      item.DocumentId,
			DocumentName:    item.DocumentName,
			LastIndexTaskId: item.LastIndexTaskId,
			Score:           item.Score,
			Reason:          item.Reason,
			Source:          item.Source,
			Features:        item.Features,
		})
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
		RaptorTopK:                int32(o.RaptorTopK),
		RaptorSourceChunkTopK:     int32(o.RaptorSourceChunkTopK),
		CandidateTopK:             int32(o.CandidateTopK),
		RerankCandidateTopK:       int32(o.RerankCandidateTopK),
		FinalTopK:                 int32(o.FinalTopK),
		RerankEnabled:             o.RerankEnabled,
		ChannelTimeoutMs:          durationToMillis(o.ChannelTimeout),
		SubQuestionTimeoutMs:      durationToMillis(o.SubQuestionTimeout),
		MinVectorSimilarity:       o.MinVectorSimilarity,
		KeywordRelativeScoreFloor: o.KeywordRelativeScoreFloor,
		KeywordChannelEnabled:     o.KeywordChannelEnabled,
		TableChannelEnabled:       o.TableChannelEnabled,
		GraphRagChannelEnabled:    o.GraphRagChannelEnabled,
		RaptorChannelEnabled:      o.RaptorChannelEnabled,
		Hybrid:                    hybridOptionsToProto(o.Hybrid),
		KbConfigConflictFields:    o.KbConfigConflictFields,
	}
}

func hybridOptionsToProto(o *vo.HybridOptions) *knowledgev1.HybridOptions {
	if o == nil {
		return nil
	}
	return &knowledgev1.HybridOptions{
		VectorWeight:        o.VectorWeight,
		KeywordWeight:       o.KeywordWeight,
		TableWeight:         o.TableWeight,
		GraphRagWeight:      o.GraphRagWeight,
		RaptorWeight:        o.RaptorWeight,
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
		Raptor: &knowledgev1.RaptorBuildOptions{
			RaptorBuildEnabled:        o.Raptor.RaptorBuildEnabled,
			RaptorMaxClusterSize:      int32(o.Raptor.RaptorMaxClusterSize),
			RaptorMaxLevels:           int32(o.Raptor.RaptorMaxLevels),
			RaptorLlmSummaryEnabled:   o.Raptor.RaptorLlmSummaryEnabled,
			RaptorSummaryQualityFloor: o.Raptor.RaptorSummaryQualityFloor,
		},
	}
}

func durationToMillis(d time.Duration) int64 {
	return int64(d / time.Millisecond)
}
