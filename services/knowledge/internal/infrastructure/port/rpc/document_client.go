package rpc

import (
	"context"

	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

type KnowledgeDocumentClient struct {
	client documentv1.DocumentServiceClient
}

func NewKnowledgeDocumentClient(client documentv1.DocumentServiceClient) *KnowledgeDocumentClient {
	return &KnowledgeDocumentClient{client: client}
}

func (c *KnowledgeDocumentClient) CountDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	resp, err := c.client.CountDocumentsByKbIds(ctx, &documentv1.CountByKbIdsRequest{KbIds: kbIds})
	if err != nil {
		return nil, err
	}
	return resp.GetCounts(), nil
}

func (c *KnowledgeDocumentClient) CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	resp, err := c.client.CountRetrievableDocumentsByKbIds(ctx, &documentv1.CountByKbIdsRequest{KbIds: kbIds})
	if err != nil {
		return nil, err
	}
	return resp.GetCounts(), nil
}

func (c *KnowledgeDocumentClient) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error) {
	resp, err := c.client.FindRetrievableByKbIds(ctx, &documentv1.FindRetrievableByKbIdsRequest{KbIds: kbIds})
	if err != nil {
		return nil, err
	}
	return protoDocumentMetadataToKnowledgeVO(resp.GetItems()), nil
}

func (c *KnowledgeDocumentClient) FindRetrieveDocumentByIds(ctx context.Context, ids ...int64) ([]*vo.DocumentMetadata, error) {
	resp, err := c.client.FindRetrieveDocumentByIds(ctx, &documentv1.FindRetrieveDocumentByIdsRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	return protoDocumentMetadataToKnowledgeVO(resp.GetItems()), nil
}

func (c *KnowledgeDocumentClient) FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*vo.DocumentProfile, error) {
	resp, err := c.client.FindDocumentProfileByDocIds(ctx, &documentv1.FindRetrieveDocumentByIdsRequest{Ids: docIds})
	if err != nil {
		return nil, err
	}
	return protoDocumentProfileToKnowledgeVO(resp.GetItems()), nil
}

func (c *KnowledgeDocumentClient) FindDocumentProfiles(ctx context.Context) ([]*vo.DocumentProfile, error) {
	resp, err := c.client.FindDocumentProfiles(ctx, &documentv1.FindDocumentProfilesRequest{})
	if err != nil {
		return nil, err
	}
	return protoDocumentProfileToKnowledgeVO(resp.GetItems()), nil
}

func protoDocumentMetadataToKnowledgeVO(items []*documentv1.DocumentMetadata) []*vo.DocumentMetadata {
	return utils.Map(items, func(m *documentv1.DocumentMetadata) *vo.DocumentMetadata {
		return &vo.DocumentMetadata{
			DocumentId:        m.GetDocumentId(),
			DocumentName:      m.GetDocumentName(),
			KnowledgeBaseId:   m.GetKnowledgeBaseId(),
			KnowledgeBaseName: m.GetKnowledgeBaseName(),
			LastIndexTaskId:   m.GetLastIndexTaskId(),
		}
	})
}

func protoDocumentProfileToKnowledgeVO(items []*documentv1.DocumentProfile) []*vo.DocumentProfile {
	return utils.Map(items, func(m *documentv1.DocumentProfile) *vo.DocumentProfile {
		return &vo.DocumentProfile{
			DocumentId:       m.GetDocumentId(),
			DocumentSummary:  m.GetDocumentSummary(),
			CoreTopics:       m.GetCoreTopics(),
			ExampleQuestions: m.GetExampleQuestions(),
			DocumentType:     m.GetDocumentType(),
		}
	})
}
