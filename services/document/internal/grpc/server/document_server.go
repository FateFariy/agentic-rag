package server

import (
	"context"
	"strconv"

	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/proto/knowagent/document/v1"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	dvo "github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
	"github.com/swiftbit/know-agent/services/document/internal/infrastructure/port/gateway"
)

// 编译期断言：DocumentServer 实现 DocumentServiceServer。
var _ documentv1.DocumentServiceServer = (*DocumentServer)(nil)

// DocumentServer document 域 gRPC 服务端，协议转换与 client/document.go 严格镜像对称。
type DocumentServer struct {
	documentv1.UnimplementedDocumentServiceServer
	adapter *gateway.DocumentAdapter
}

// NewDocumentServer 创建 document gRPC 服务端。
func NewDocumentServer(adapter *gateway.DocumentAdapter) *DocumentServer {
	return &DocumentServer{adapter: adapter}
}

func (s *DocumentServer) FindRetrievableByKbIds(ctx context.Context, req *documentv1.FindRetrievableByKbIdsRequest) (*documentv1.DocumentMetadataList, error) {
	items, err := s.adapter.FindRetrievableByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.DocumentMetadataList{Items: metadataListToProto(items)}, nil
}

func (s *DocumentServer) FindRetrieveDocumentByIds(ctx context.Context, req *documentv1.FindRetrieveDocumentByIdsRequest) (*documentv1.DocumentMetadataList, error) {
	items, err := s.adapter.FindRetrieveDocumentByIds(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.DocumentMetadataList{Items: metadataListToProto(items)}, nil
}

func (s *DocumentServer) FindParentChunks(ctx context.Context, req *documentv1.FindParentChunksRequest) (*documentv1.DocumentChunkList, error) {
	chunks, err := s.adapter.FindParentChunks(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv1.DocumentChunk, 0, len(chunks))
	for _, chunk := range chunks {
		items = append(items, parentChunkToProto(chunk))
	}
	return &documentv1.DocumentChunkList{Items: items}, nil
}

func (s *DocumentServer) ListSections(ctx context.Context, req *documentv1.ListSectionsRequest) (*documentv1.GraphSectionList, error) {
	nodes, err := s.adapter.ListSections(ctx, req.GetDocumentId())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv1.GraphSection, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, sectionToProto(node))
	}
	return &documentv1.GraphSectionList{Items: items}, nil
}

func (s *DocumentServer) FindSectionById(ctx context.Context, req *documentv1.FindSectionByIdRequest) (*documentv1.GraphSectionOrEmpty, error) {
	node, err := s.adapter.FindSectionById(ctx, req.GetDocumentId(), req.GetNodeId())
	if err != nil {
		return nil, err
	}
	return &documentv1.GraphSectionOrEmpty{Section: sectionToProto(node)}, nil
}

func (s *DocumentServer) FindSectionByCode(ctx context.Context, req *documentv1.FindSectionByCodeRequest) (*documentv1.GraphSectionOrEmpty, error) {
	node, err := s.adapter.FindSectionByCode(ctx, req.GetDocumentId(), req.GetSectionCode())
	if err != nil {
		return nil, err
	}
	return &documentv1.GraphSectionOrEmpty{Section: sectionToProto(node)}, nil
}

func (s *DocumentServer) CountDocumentsByKbIds(ctx context.Context, req *documentv1.CountByKbIdsRequest) (*documentv1.Int64CountMap, error) {
	counts, err := s.adapter.CountDocumentsByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.Int64CountMap{Counts: counts}, nil
}

func (s *DocumentServer) CountRetrievableDocumentsByKbIds(ctx context.Context, req *documentv1.CountByKbIdsRequest) (*documentv1.Int64CountMap, error) {
	counts, err := s.adapter.CountRetrievableDocumentsByKbIds(ctx, req.GetKbIds())
	if err != nil {
		return nil, err
	}
	return &documentv1.Int64CountMap{Counts: counts}, nil
}

func (s *DocumentServer) FindDocumentProfileByDocIds(ctx context.Context, req *documentv1.FindRetrieveDocumentByIdsRequest) (*documentv1.DocumentProfileList, error) {
	profiles, err := s.adapter.FindDocumentProfileByDocIds(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	items := make([]*documentv1.DocumentProfile, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, profileToProto(profile))
	}
	return &documentv1.DocumentProfileList{Items: items}, nil
}

func (s *DocumentServer) FindDocumentProfiles(ctx context.Context, _ *documentv1.FindDocumentProfilesRequest) (*documentv1.DocumentProfileList, error) {
	profiles, err := s.adapter.FindDocumentProfiles(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*documentv1.DocumentProfile, 0, len(profiles))
	for _, profile := range profiles {
		items = append(items, profileToProto(profile))
	}
	return &documentv1.DocumentProfileList{Items: items}, nil
}

// ==================== 私有转换（与 grpc/client 端映射反向对称） ====================

func metadataListToProto(items []*dvo.DocumentMetadata) []*documentv1.DocumentMetadata {
	out := make([]*documentv1.DocumentMetadata, 0, len(items))
	for _, m := range items {
		out = append(out, metadataToProto(m))
	}
	return out
}

func metadataToProto(m *dvo.DocumentMetadata) *documentv1.DocumentMetadata {
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

// TODO(阶段4): 当前仅填充父块基础字段；proto.DocumentChunk 中的评分/rerank 等 32 个字段
// 由检索链路（chat 侧）填充，父块召回路径本就不产生这些值，与原进程内适配器行为一致。
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
