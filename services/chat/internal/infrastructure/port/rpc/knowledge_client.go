package rpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/conversation"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/vo"
)

// KnowledgeClient 实现 chat 域出站消费 knowledge 服务的接口：
//   - adapter.KnowledgeBaseGateway（DetermineKnowledgeScope）
//   - conversation.KnowledgeRouter（Route / RecordShadowRoute）
//
// document 域消费 knowledge 的客户端在 services/document/internal/grpcclient。
type KnowledgeClient struct {
	client knowledgev1.KnowledgeServiceClient
}

func NewKnowledgeClient(client knowledgev1.KnowledgeServiceClient) *KnowledgeClient {
	return &KnowledgeClient{client: client}
}

func (c *KnowledgeClient) DetermineKnowledgeScope(ctx context.Context, selectMode string, kbIds []string) (*vo.KnowledgeBaseSelectionSnapshot, error) {
	resp, err := c.client.DetermineKnowledgeScope(ctx, &knowledgev1.DetermineKnowledgeScopeRequest{SelectMode: selectMode, KbIds: kbIds})
	if err != nil {
		return nil, err
	}
	return protoSnapshotToChatVO(resp), nil
}

func (c *KnowledgeClient) Route(ctx context.Context, input *conversation.KnowledgeRouteInput) (*vo.KnowledgeRouteDecision, error) {
	resp, err := c.client.Route(ctx, &knowledgev1.RouteRequest{Input: knowledgeRouteInputToProto(input)})
	if err != nil {
		return nil, err
	}
	return protoDecisionToChatVO(resp), nil
}

func (c *KnowledgeClient) RecordShadowRoute(ctx context.Context, input *conversation.KnowledgeRouteInput) error {
	_, err := c.client.RecordShadowRoute(ctx, &knowledgev1.RecordShadowRouteRequest{Input: knowledgeRouteInputToProto(input)})
	return err
}

func knowledgeRouteInputToProto(p *conversation.KnowledgeRouteInput) *knowledgev1.KnowledgeRouteInput {
	if p == nil {
		return nil
	}
	return &knowledgev1.KnowledgeRouteInput{
		ConversationId:             p.ConversationId,
		ExchangeId:                 p.ExchangeId,
		Question:                   p.Question,
		RewriteQuestion:            p.RewriteQuestion,
		KnowledgeBaseSelectionMode: p.KnowledgeBaseSelectionMode,
		SelectedDocumentId:         p.SelectedDocumentId,
		SelectedKnowledgeBaseIds:   p.SelectedKnowledgeBaseIds,
		SelectedKnowledgeBaseNames: p.SelectedKnowledgeBaseNames,
		AllowedDocumentIds:         p.AllowedDocumentIds,
	}
}

func protoSnapshotToChatVO(p *knowledgev1.KnowledgeBaseSelectionSnapshot) *vo.KnowledgeBaseSelectionSnapshot {
	if p == nil {
		return nil
	}
	return &vo.KnowledgeBaseSelectionSnapshot{
		SelectionMode:              enum.KnowledgeBaseSelectionMode(p.GetSelectionMode()),
		SelectedKnowledgeBaseIds:   p.GetSelectedKnowledgeBaseIds(),
		SelectedKnowledgeBaseNames: p.GetSelectedKnowledgeBaseNames(),
		SelectedKnowledgeBases:     protoKnowledgeBaseListToChatVO(p.GetSelectedKnowledgeBases()),
		AllowedDocuments:           protoDocumentMetadataListToChatVO(p.GetAllowedDocuments()),
		RagRuntimeOptions:          protoRagRuntimeOptionsToChatVO(p.GetRagRuntimeOptions()),
	}
}

func protoKnowledgeBaseListToChatVO(items []*knowledgev1.KnowledgeBase) []*vo.KnowledgeBase {
	out := make([]*vo.KnowledgeBase, 0, len(items))
	for _, m := range items {
		out = append(out, &vo.KnowledgeBase{
			ID:                  m.GetId(),
			BaseName:            m.GetBaseName(),
			Description:         m.GetDescription(),
			EmbeddingModel:      m.GetEmbeddingModel(),
			RetrievalConfigJson: json.RawMessage(m.GetRetrievalConfigJson()),
			GraphRagConfigJson:  json.RawMessage(m.GetGraphRagConfigJson()),
			MetadataFilterJson:  json.RawMessage(m.GetMetadataFilterJson()),
			IsDefault:           int(m.GetIsDefault()),
			SortOrder:           int(m.GetSortOrder()),
		})
	}
	return out
}

func protoDocumentMetadataListToChatVO(items []*documentv1.DocumentMetadata) []*vo.DocumentMetadata {
	out := make([]*vo.DocumentMetadata, 0, len(items))
	for _, m := range items {
		out = append(out, &vo.DocumentMetadata{
			DocumentId:        m.GetDocumentId(),
			DocumentName:      m.GetDocumentName(),
			KnowledgeBaseId:   m.GetKnowledgeBaseId(),
			KnowledgeBaseName: m.GetKnowledgeBaseName(),
			LastIndexTaskId:   m.GetLastIndexTaskId(),
		})
	}
	return out
}

func protoRagRuntimeOptionsToChatVO(p *knowledgev1.RagRuntimeOptions) *vo.RagRuntimeOptions {
	if p == nil {
		return nil
	}
	return &vo.RagRuntimeOptions{
		VectorTopK:                int(p.GetVectorTopK()),
		KeywordTopK:               int(p.GetKeywordTopK()),
		GraphRagTopK:              int(p.GetGraphRagTopK()),
		GraphRagMaxHops:           int(p.GetGraphRagMaxHops()),
		CandidateTopK:             int(p.GetCandidateTopK()),
		RerankCandidateTopK:       int(p.GetRerankCandidateTopK()),
		FinalTopK:                 int(p.GetFinalTopK()),
		RerankEnabled:             p.GetRerankEnabled(),
		ChannelTimeout:            time.Duration(p.GetChannelTimeoutMs()) * time.Millisecond,
		SubQuestionTimeout:        time.Duration(p.GetSubQuestionTimeoutMs()) * time.Millisecond,
		MinVectorSimilarity:       p.GetMinVectorSimilarity(),
		KeywordRelativeScoreFloor: p.GetKeywordRelativeScoreFloor(),
		KeywordChannelEnabled:     p.GetKeywordChannelEnabled(),
		GraphRagChannelEnabled:    p.GetGraphRagChannelEnabled(),
		Hybrid:                    protoHybridOptionsToChatVO(p.GetHybrid()),
		KbConfigConflictFields:    p.GetKbConfigConflictFields(),
	}
}

func protoHybridOptionsToChatVO(p *knowledgev1.HybridOptions) *vo.HybridOptions {
	if p == nil {
		return nil
	}
	return &vo.HybridOptions{
		VectorWeight:        p.GetVectorWeight(),
		KeywordWeight:       p.GetKeywordWeight(),
		GraphRagWeight:      p.GetGraphRagWeight(),
		RankWeight:          p.GetRankWeight(),
		OriginalScoreWeight: p.GetOriginalScoreWeight(),
		MetadataBoostWeight: p.GetMetadataBoostWeight(),
		MaxMetadataBoost:    p.GetMaxMetadataBoost(),
	}
}

func protoDecisionToChatVO(p *knowledgev1.KnowledgeRouteDecision) *vo.KnowledgeRouteDecision {
	if p == nil {
		return nil
	}
	return &vo.KnowledgeRouteDecision{
		Scopes:          protoScopeCandidatesToChatVO(p.GetScopes()),
		Topics:          protoTopicCandidatesToChatVO(p.GetTopics()),
		Documents:       protoDocumentCandidatesToChatVO(p.GetDocuments()),
		Confidence:      p.GetConfidence(),
		RouteStatus:     p.GetRouteStatus(),
		Reason:          p.GetReason(),
		Source:          p.GetSource(),
		Degraded:        p.GetDegraded(),
		DegradedReasons: p.GetDegradedReasons(),
	}
}

func protoScopeCandidatesToChatVO(items []*knowledgev1.ScopeRouteCandidate) []*vo.ScopeRouteCandidate {
	out := make([]*vo.ScopeRouteCandidate, 0, len(items))
	for _, m := range items {
		out = append(out, &vo.ScopeRouteCandidate{
			ScopeId:   m.GetScopeId(),
			ScopeName: m.GetScopeName(),
			Score:     m.GetScore(),
			Reason:    m.GetReason(),
			Source:    m.GetSource(),
			Features:  m.GetFeatures(),
		})
	}
	return out
}

func protoTopicCandidatesToChatVO(items []*knowledgev1.TopicRouteCandidate) []*vo.TopicRouteCandidate {
	out := make([]*vo.TopicRouteCandidate, 0, len(items))
	for _, m := range items {
		out = append(out, &vo.TopicRouteCandidate{
			TopicId:   m.GetTopicId(),
			TopicName: m.GetTopicName(),
			ScopeId:   m.GetScopeId(),
			Score:     m.GetScore(),
			Reason:    m.GetReason(),
			Source:    m.GetSource(),
			Features:  m.GetFeatures(),
		})
	}
	return out
}

func protoDocumentCandidatesToChatVO(items []*knowledgev1.DocumentRouteCandidate) []*vo.DocumentRouteCandidate {
	out := make([]*vo.DocumentRouteCandidate, 0, len(items))
	for _, m := range items {
		out = append(out, &vo.DocumentRouteCandidate{
			DocumentId:      m.GetDocumentId(),
			DocumentName:    m.GetDocumentName(),
			LastIndexTaskId: m.GetLastIndexTaskId(),
			Score:           m.GetScore(),
			Reason:          m.GetReason(),
			Source:          m.GetSource(),
			Features:        m.GetFeatures(),
		})
	}
	return out
}
