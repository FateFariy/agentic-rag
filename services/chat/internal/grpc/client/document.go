package client

import (
	"context"

	"google.golang.org/grpc"

	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/proto/knowagent/document/v1"
	
)

// ChatDocumentClient 实现 chat 域消费 document 的两个接口：
//   - adapter.DocumentGateway（FindRetrievableByKbIds / FindRetrieveDocumentByIds / FindParentChunks，返回 chat vo）
//   - route.GraphQuerier（ListSections / FindSectionById / FindSectionByCode，返回 chat vo）
type ChatDocumentClient struct {
	client documentv1.DocumentServiceClient
}

func NewChatDocumentClient(cc *grpc.ClientConn) *ChatDocumentClient {
	return &ChatDocumentClient{client: documentv1.NewDocumentServiceClient(cc)}
}

func (c *ChatDocumentClient) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error) {
	resp, err := c.client.FindRetrievableByKbIds(ctx, &documentv1.FindRetrievableByKbIdsRequest{KbIds: kbIds})
	if err != nil {
		return nil, err
	}
	return protoDocumentMetadataToChatVO(resp.GetItems()), nil
}

func (c *ChatDocumentClient) FindRetrieveDocumentByIds(ctx context.Context, ids ...int64) ([]*vo.DocumentMetadata, error) {
	resp, err := c.client.FindRetrieveDocumentByIds(ctx, &documentv1.FindRetrieveDocumentByIdsRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	return protoDocumentMetadataToChatVO(resp.GetItems()), nil
}

func (c *ChatDocumentClient) FindParentChunks(ctx context.Context, ids []int64) ([]*vo.DocumentChunk, error) {
	resp, err := c.client.FindParentChunks(ctx, &documentv1.FindParentChunksRequest{Ids: ids})
	if err != nil {
		return nil, err
	}
	out := make([]*vo.DocumentChunk, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		out = append(out, protoChunkToChatVO(item))
	}
	return out, nil
}

func (c *ChatDocumentClient) ListSections(ctx context.Context, documentId int64) ([]*vo.GraphSection, error) {
	resp, err := c.client.ListSections(ctx, &documentv1.ListSectionsRequest{DocumentId: documentId})
	if err != nil {
		return nil, err
	}
	return protoSectionsToChatVO(resp.GetItems()), nil
}

func (c *ChatDocumentClient) FindSectionById(ctx context.Context, documentId int64, nodeId int64) (*vo.GraphSection, error) {
	resp, err := c.client.FindSectionById(ctx, &documentv1.FindSectionByIdRequest{DocumentId: documentId, NodeId: nodeId})
	if err != nil {
		return nil, err
	}
	return protoSectionToChatVO(resp.GetSection()), nil
}

func (c *ChatDocumentClient) FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*vo.GraphSection, error) {
	resp, err := c.client.FindSectionByCode(ctx, &documentv1.FindSectionByCodeRequest{DocumentId: documentId, SectionCode: sectionCode})
	if err != nil {
		return nil, err
	}
	return protoSectionToChatVO(resp.GetSection()), nil
}

func protoDocumentMetadataToChatVO(items []*documentv1.DocumentMetadata) []*vo.DocumentMetadata {
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

func protoSectionsToChatVO(items []*documentv1.GraphSection) []*vo.GraphSection {
	out := make([]*vo.GraphSection, 0, len(items))
	for _, m := range items {
		out = append(out, protoSectionToChatVO(m))
	}
	return out
}

func protoSectionToChatVO(m *documentv1.GraphSection) *vo.GraphSection {
	if m == nil {
		return nil
	}
	return &vo.GraphSection{
		NodeId:            m.GetNodeId(),
		DocumentId:        m.GetDocumentId(),
		ParseTaskId:       m.GetParseTaskId(),
		NodeNo:            int(m.GetNodeNo()),
		Depth:             int(m.GetDepth()),
		ParentNodeId:      m.GetParentNodeId(),
		PrevSiblingNodeId: m.GetPrevSiblingNodeId(),
		NextSiblingNodeId: m.GetNextSiblingNodeId(),
		NodeCode:          m.GetNodeCode(),
		Title:             m.GetTitle(),
		AnchorText:        m.GetAnchorText(),
		SectionPath:       m.GetSectionPath(),
		CanonicalPath:     m.GetCanonicalPath(),
		ContentText:       m.GetContentText(),
	}
}

func protoChunkToChatVO(m *documentv1.DocumentChunk) *vo.DocumentChunk {
	if m == nil {
		return nil
	}
	return &vo.DocumentChunk{
		ID:                  m.GetId(),
		Score:               m.GetScore(),
		Content:             m.GetContent(),
		SourceType:          m.GetSourceType(),
		Channel:             m.GetChannel(),
		TaskId:              m.GetTaskId(),
		ParentChunkId:       m.GetParentChunkId(),
		DocumentId:          m.GetDocumentId(),
		ChunkNo:             int(m.GetChunkNo()),
		SectionPath:         m.GetSectionPath(),
		StructureNodeId:     m.GetStructureNodeId(),
		StructureNodeType:   int(m.GetStructureNodeType()),
		CanonicalPath:       m.GetCanonicalPath(),
		ItemIndex:           int(m.GetItemIndex()),
		OriginalSnippet:     m.GetOriginalSnippet(),
		DocumentName:        m.GetDocumentName(),
		KnowledgeBaseId:     m.GetKnowledgeBaseId(),
		KnowledgeBaseName:   m.GetKnowledgeBaseName(),
		IsElevated:          int(m.GetIsElevated()),
		MetadataBoost:       m.GetMetadataBoost(),
		NormalizedScore:     m.GetNormalizedScore(),
		RRFScore:            m.GetRrfScore(),
		RerankScore:         m.GetRerankScore(),
		ParentChunkNo:       int(m.GetParentChunkNo()),
		Title:               m.GetTitle(),
		Url:                 m.GetUrl(),
		ToolName:            m.GetToolName(),
		RerankModel:         m.GetRerankModel(),
		RerankQuery:         m.GetRerankQuery(),
		RerankDurationMs:    m.GetRerankDurationMs(),
		RerankOriginalIndex: int(m.GetRerankOriginalIndex()),
		Extra:               m.GetExtra().AsMap(),
	}
}
