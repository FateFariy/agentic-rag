package grpc

import (
	"context"
	"strconv"

	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// 编译期断言：DocumentGrpcServer 实现 DocumentServiceServer
var _ documentv1.DocumentServiceServer = (*DocumentGrpcServer)(nil)

// DocumentGrpcServer documentv1 域 gRPC 服务端
type DocumentGrpcServer struct {
	documentv1.UnimplementedDocumentServiceServer
	query logic.DocumentQueryLogic
}

func NewDocumentGrpcServer(query logic.DocumentQueryLogic) *DocumentGrpcServer {
	return &DocumentGrpcServer{query: query}
}

func (s *DocumentGrpcServer) FindRetrievableByKbIds(ctx context.Context, req *documentv1.FindRetrievableByKbIdsRequest) (*documentv1.DocumentMetadataList, error) {
	items, err := s.query.FindRetrievableByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.DocumentMetadataList{Items: metadataListToProto(items)}, nil
}

func (s *DocumentGrpcServer) FindRetrieveDocumentByIds(ctx context.Context, req *documentv1.FindRetrieveDocumentByIdsRequest) (*documentv1.DocumentMetadataList, error) {
	items, err := s.query.FindRetrieveDocumentByIds(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.DocumentMetadataList{Items: metadataListToProto(items)}, nil
}

func (s *DocumentGrpcServer) FindParentChunks(ctx context.Context, req *documentv1.FindParentChunksRequest) (*documentv1.DocumentChunkList, error) {
	chunks, err := s.query.FindParentChunks(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv1.DocumentChunk, 0, len(chunks))
	for _, chunk := range chunks {
		items = append(items, parentChunkToProto(chunk))
	}
	return &documentv1.DocumentChunkList{Items: items}, nil
}

func (s *DocumentGrpcServer) ListSections(ctx context.Context, req *documentv1.ListSectionsRequest) (*documentv1.GraphSectionList, error) {
	nodes, err := s.query.ListSections(ctx, req.GetDocumentId())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv1.GraphSection, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, sectionToProto(node))
	}
	return &documentv1.GraphSectionList{Items: items}, nil
}

func (s *DocumentGrpcServer) FindSectionById(ctx context.Context, req *documentv1.FindSectionByIdRequest) (*documentv1.GraphSectionOrEmpty, error) {
	node, err := s.query.FindSectionById(ctx, req.GetDocumentId(), req.GetNodeId())
	if err != nil {
		return nil, err
	}
	return &documentv1.GraphSectionOrEmpty{Section: sectionToProto(node)}, nil
}

func (s *DocumentGrpcServer) FindSectionByCode(ctx context.Context, req *documentv1.FindSectionByCodeRequest) (*documentv1.GraphSectionOrEmpty, error) {
	node, err := s.query.FindSectionByCode(ctx, req.GetDocumentId(), req.GetSectionCode())
	if err != nil {
		return nil, err
	}
	return &documentv1.GraphSectionOrEmpty{Section: sectionToProto(node)}, nil
}

func (s *DocumentGrpcServer) CountDocumentsByKbIds(ctx context.Context, req *documentv1.CountByKbIdsRequest) (*documentv1.Int64CountMap, error) {
	counts, err := s.query.CountDocumentsByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.Int64CountMap{Counts: counts}, nil
}

func (s *DocumentGrpcServer) CountRetrievableDocumentsByKbIds(ctx context.Context, req *documentv1.CountByKbIdsRequest) (*documentv1.Int64CountMap, error) {
	counts, err := s.query.CountRetrievableDocumentsByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.Int64CountMap{Counts: counts}, nil
}

func (s *DocumentGrpcServer) FindDocumentProfileByDocIds(ctx context.Context, req *documentv1.FindRetrieveDocumentByIdsRequest) (*documentv1.DocumentProfileList, error) {
	profiles, err := s.query.FindDocumentProfileByDocIds(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	items := utils.Map(profiles, func(profile *entity.DocumentProfile) *documentv1.DocumentProfile {
		return profileToProto(profile)
	})
	return &documentv1.DocumentProfileList{Items: items}, nil
}

func (s *DocumentGrpcServer) FindDocumentProfiles(ctx context.Context, _ *documentv1.FindDocumentProfilesRequest) (*documentv1.DocumentProfileList, error) {
	profiles, err := s.query.FindDocumentProfiles(ctx)
	if err != nil {
		return nil, err
	}
	items := utils.Map(profiles, func(profile *entity.DocumentProfile) *documentv1.DocumentProfile {
		return profileToProto(profile)
	})
	return &documentv1.DocumentProfileList{Items: items}, nil
}

// ==================== 私有转换（与 grpc/client 端映射反向对称） ====================

func metadataListToProto(items []*vo.DocumentMetadata) []*documentv1.DocumentMetadata {
	return utils.Map(items, func(item *vo.DocumentMetadata) *documentv1.DocumentMetadata {
		return metadataToProto(item)
	})
}

func metadataToProto(m *vo.DocumentMetadata) *documentv1.DocumentMetadata {
	if m == nil {
		return nil
	}
	return &documentv1.DocumentMetadata{
		DocumentId:        m.DocumentId,
		DocumentName:      m.DocumentName,
		KnowledgeBaseId:   m.KnowledgeBaseId,
		KnowledgeBaseName: m.KnowledgeBaseName,
		LastIndexTaskId:   m.LastIndexTaskId,
	}
}

func parentChunkToProto(chunk *entity.DocumentParentChunk) *documentv1.DocumentChunk {
	if chunk == nil {
		return nil
	}
	return &documentv1.DocumentChunk{
		Id:                strconv.FormatInt(chunk.ID, 10),
		Content:           chunk.ParentText,
		SourceType:        chunk.SourceTypeName,
		TaskId:            chunk.TaskId,
		DocumentId:        chunk.DocumentId,
		ChunkNo:           int32(chunk.ParentNo),
		SectionPath:       chunk.SectionPath,
		StructureNodeId:   chunk.StructureNodeId,
		StructureNodeType: int32(chunk.StructureNodeType),
		CanonicalPath:     chunk.CanonicalPath,
		ItemIndex:         int32(chunk.ItemIndex),
		OriginalSnippet:   chunk.ParentText,
	}
}

func sectionToProto(node *entity.StructureNode) *documentv1.GraphSection {
	if node == nil {
		return nil
	}
	return &documentv1.GraphSection{
		NodeId:            node.ID,
		DocumentId:        node.DocumentId,
		ParseTaskId:       node.ParseTaskId,
		NodeNo:            int32(node.NodeNo),
		Depth:             int32(node.Depth),
		ParentNodeId:      node.ParentNodeId,
		PrevSiblingNodeId: node.PrevSiblingNodeId,
		NextSiblingNodeId: node.NextSiblingNodeId,
		NodeCode:          node.NodeCode,
		Title:             node.Title,
		AnchorText:        node.AnchorText,
		SectionPath:       node.SectionPath,
		CanonicalPath:     node.CanonicalPath,
		ContentText:       node.ContentText,
	}
}

func profileToProto(profile *entity.DocumentProfile) *documentv1.DocumentProfile {
	if profile == nil {
		return nil
	}
	return &documentv1.DocumentProfile{
		DocumentId:       profile.DocumentId,
		DocumentSummary:  profile.DocumentSummary,
		CoreTopics:       profile.CoreTopics,
		ExampleQuestions: profile.ExampleQuestions,
		DocumentType:     profile.DocumentType,
	}
}
