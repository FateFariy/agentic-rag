package gateway

import (
	"context"

	"google.golang.org/grpc"

	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/proto/knowagent/document/v1"
	kvo "github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

// KnowledgeDocumentClient 实现 knowledge 域消费 document 服务的出站接口（返回 knowledge vo）。
type KnowledgeDocumentClient struct {
	client documentv1.DocumentServiceClient
}

func NewKnowledgeDocumentClient(cc *grpc.ClientConn) *KnowledgeDocumentClient {
	return &KnowledgeDocumentClient{client: documentv1.NewDocumentServiceClient(cc)}
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

func (c *KnowledgeDocumentClient) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*kvo.DocumentMetadata, error) {
	resp, err := c.client.FindRetrievableByKbIds(ctx, &documentv1.FindRetrievableByKbIdsRequest{KbIds: kbIds})
	if err != nil {
		return nil, err
	}
	return protoDocumentMetadataToKnowledgeVO(resp.GetItems()), nil
}

func (c *KnowledgeDocumentClient) FindRetrieveDocumentByIds(ctx context.Context, ids ...int64) ([]*kvo.DocumentMetadata, error) {
	resp, err := c.client.FindRetrieveDocumentByIds(ctx, &documentv1.FindRetrieveDocumentByIdsRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	return protoDocumentMetadataToKnowledgeVO(resp.GetItems()), nil
}

func (c *KnowledgeDocumentClient) FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*kvo.DocumentProfile, error) {
	resp, err := c.client.FindDocumentProfileByDocIds(ctx, &documentv1.FindRetrieveDocumentByIdsRequest{Ids: docIds})
	if err != nil {
		return nil, err
	}
	return protoDocumentProfileToKnowledgeVO(resp.GetItems()), nil
}

func (c *KnowledgeDocumentClient) FindDocumentProfiles(ctx context.Context) ([]*kvo.DocumentProfile, error) {
	resp, err := c.client.FindDocumentProfiles(ctx, &documentv1.FindDocumentProfilesRequest{})
	if err != nil {
		return nil, err
	}
	return protoDocumentProfileToKnowledgeVO(resp.GetItems()), nil
}

func protoDocumentMetadataToKnowledgeVO(items []*documentv1.DocumentMetadata) []*kvo.DocumentMetadata {
	out := make([]*kvo.DocumentMetadata, 0, len(items))
	for _, m := range items {
		out = append(out, &kvo.DocumentMetadata{
			DocumentId:        m.GetDocumentId(),
			DocumentName:      m.GetDocumentName(),
			KnowledgeBaseId:   m.GetKnowledgeBaseId(),
			KnowledgeBaseName: m.GetKnowledgeBaseName(),
			LastIndexTaskId:   m.GetLastIndexTaskId(),
		})
	}
	return out
}

func protoDocumentProfileToKnowledgeVO(items []*documentv1.DocumentProfile) []*kvo.DocumentProfile {
	out := make([]*kvo.DocumentProfile, 0, len(items))
	for _, m := range items {
		out = append(out, &kvo.DocumentProfile{
			DocumentId:       m.GetDocumentId(),
			DocumentSummary:  m.GetDocumentSummary(),
			CoreTopics:       m.GetCoreTopics(),
			ExampleQuestions: m.GetExampleQuestions(),
			DocumentType:     m.GetDocumentType(),
		})
	}
	return out
}
