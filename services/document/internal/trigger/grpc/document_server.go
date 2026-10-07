package grpc

import (
	"context"
	"strconv"

	documentv2 "github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	dvo "github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// 编译期断言：DocumentGrpcServer 实现 DocumentServiceServer。
var _ documentv2.DocumentServiceServer = (*DocumentGrpcServer)(nil)

// DocumentGrpcServer document 域 gRPC 服务端，只承担 proto 协议转换，业务逻辑由领域层提供。
type DocumentGrpcServer struct {
	documentv2.UnimplementedDocumentServiceServer
	query logic.DocumentQueryLogic
}

// NewDocumentGrpcServer 创建 document gRPC 服务端。
func NewDocumentGrpcServer(query logic.DocumentQueryLogic) *DocumentGrpcServer {
	return &DocumentGrpcServer{query: query}
}

func (s *DocumentGrpcServer) FindRetrievableByKbIds(ctx context.Context, req *documentv2.FindRetrievableByKbIdsRequest) (*documentv2.DocumentMetadataList, error) {
	items, err := s.query.FindRetrievableByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv2.DocumentMetadataList{Items: metadataListToProto(items)}, nil
}

func (s *DocumentGrpcServer) FindRetrieveDocumentByIds(ctx context.Context, req *documentv2.FindRetrieveDocumentByIdsRequest) (*documentv2.DocumentMetadataList, error) {
	items, err := s.query.FindRetrieveDocumentByIds(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	return &documentv2.DocumentMetadataList{Items: metadataListToProto(items)}, nil
}

func (s *DocumentGrpcServer) FindParentChunks(ctx context.Context, req *documentv2.FindParentChunksRequest) (*documentv2.DocumentChunkList, error) {
	chunks, err := s.query.FindParentChunks(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv2.DocumentChunk, 0, len(chunks))
	for _, chunk := range chunks {
		items = append(items, parentChunkToProto(chunk))
	}
	return &documentv2.DocumentChunkList{Items: items}, nil
}

func (s *DocumentGrpcServer) ListSections(ctx context.Context, req *documentv2.ListSectionsRequest) (*documentv2.GraphSectionList, error) {
	nodes, err := s.query.ListSections(ctx, req.GetDocumentId())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv2.GraphSection, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, sectionToProto(node))
	}
	return &documentv2.GraphSectionList{Items: items}, nil
}

func (s *DocumentGrpcServer) FindSectionById(ctx context.Context, req *documentv2.FindSectionByIdRequest) (*documentv2.GraphSectionOrEmpty, error) {
	node, err := s.query.FindSectionById(ctx, req.GetDocumentId(), req.GetNodeId())
	if err != nil {
		return nil, err
	}
	return &documentv2.GraphSectionOrEmpty{Section: sectionToProto(node)}, nil
}

func (s *DocumentGrpcServer) FindSectionByCode(ctx context.Context, req *documentv2.FindSectionByCodeRequest) (*documentv2.GraphSectionOrEmpty, error) {
	node, err := s.query.FindSectionByCode(ctx, req.GetDocumentId(), req.GetSectionCode())
	if err != nil {
		return nil, err
	}
	return &documentv2.GraphSectionOrEmpty{Section: sectionToProto(node)}, nil
}

func (s *DocumentGrpcServer) CountDocumentsByKbIds(ctx context.Context, req *documentv2.CountByKbIdsRequest) (*documentv2.Int64CountMap, error) {
	counts, err := s.query.CountDocumentsByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv2.Int64CountMap{Counts: counts}, nil
}

func (s *DocumentGrpcServer) CountRetrievableDocumentsByKbIds(ctx context.Context, req *documentv2.CountByKbIdsRequest) (*documentv2.Int64CountMap, error) {
	counts, err := s.query.CountRetrievableDocumentsByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv2.Int64CountMap{Counts: counts}, nil
}

func (s *DocumentGrpcServer) FindDocumentProfileByDocIds(ctx context.Context, req *documentv2.FindRetrieveDocumentByIdsRequest) (*documentv2.DocumentProfileList, error) {
	profiles, err := s.query.FindDocumentProfileByDocIds(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv2.DocumentProfile, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, profileToProto(profile))
	}
	return &documentv2.DocumentProfileList{Items: items}, nil
}

func (s *DocumentGrpcServer) FindDocumentProfiles(ctx context.Context, _ *documentv2.FindDocumentProfilesRequest) (*documentv2.DocumentProfileList, error) {
	profiles, err := s.query.FindDocumentProfiles(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*documentv2.DocumentProfile, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, profileToProto(profile))
	}
	return &documentv2.DocumentProfileList{Items: items}, nil
}

// ==================== 私有转换（与 grpc/client 端映射反向对称） ====================

func metadataListToProto(items []*dvo.DocumentMetadata) []*documentv2.DocumentMetadata {
	out := make([]*documentv2.DocumentMetadata, 0, len(items))
	for _, m := range items {
		out = append(out, metadataToProto(m))
	}
	return out
}

func metadataToProto(m *dvo.DocumentMetadata) *documentv2.DocumentMetadata {
	if m == nil {
		return nil
	}
	return &documentv2.DocumentMetadata{
		DocumentId:        m.DocumentId,
		DocumentName:      m.DocumentName,
		KnowledgeBaseId:   m.KnowledgeBaseId,
		KnowledgeBaseName: m.KnowledgeBaseName,
		LastIndexTaskId:   m.LastIndexTaskId,
	}
}

// TODO(阶段4): 当前仅填充父块基础字段；proto.DocumentChunk 中的评分/rerank 等 32 个字段
// 由检索链路（chat 侧）填充，父块召回路径本就不产生这些值，与原进程内适配器行为一致。
func parentChunkToProto(chunk *entity.DocumentParentChunk) *documentv2.DocumentChunk {
	if chunk == nil {
		return nil
	}
	return &documentv2.DocumentChunk{
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

func sectionToProto(node *entity.StructureNode) *documentv2.GraphSection {
	if node == nil {
		return nil
	}
	return &documentv2.GraphSection{
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

func profileToProto(profile *entity.DocumentProfile) *documentv2.DocumentProfile {
	if profile == nil {
		return nil
	}
	return &documentv2.DocumentProfile{
		DocumentId:       profile.DocumentId,
		DocumentSummary:  profile.DocumentSummary,
		CoreTopics:       profile.CoreTopics,
		ExampleQuestions: profile.ExampleQuestions,
		DocumentType:     profile.DocumentType,
	}
}
