package gateway

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/services/document/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
	dvo "github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// TODO(阶段4): 本适配器是 document 服务对外 gRPC 契约的服务端入站适配器，
// 微服务目录重组时随 document 服务迁入 services/document/internal/gateway。

// DocumentAdapter document 域 gRPC 服务端的入站适配器：
// 只委托 document 仓储返回 document 域自身类型，proto 转换由 grpc/server 层负责。
type DocumentAdapter struct {
	repo adapter.DocumentRepository
}

// NewDocumentAdapter 创建 document gRPC 服务端适配器。
func NewDocumentAdapter(repo adapter.DocumentRepository) *DocumentAdapter {
	return &DocumentAdapter{repo: repo}
}

// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据。
func (a *DocumentAdapter) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*dvo.DocumentMetadata, error) {
	return a.repo.SelectRetrievableDocumentsByKbIds(ctx, kbIds...)
}

// FindRetrieveDocumentByIds 根据文档ID列表查询可检索的文档元数据。
func (a *DocumentAdapter) FindRetrieveDocumentByIds(ctx context.Context, ids []int64) ([]*dvo.DocumentMetadata, error) {
	return a.repo.SelectRetrievableDocumentsByIds(ctx, ids...)
}

// FindParentChunks 根据父块ID列表查询父块。
func (a *DocumentAdapter) FindParentChunks(ctx context.Context, ids []int64) ([]*entity.DocumentParentChunk, error) {
	return a.repo.SelectParentChunkListByIds(ctx, ids)
}

// ListSections 查询指定文档的章节结构节点（仅保留章节类型节点）。
func (a *DocumentAdapter) ListSections(ctx context.Context, documentId int64) ([]*entity.StructureNode, error) {
	nodes, err := a.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	sections := make([]*entity.StructureNode, 0, len(nodes))
	for _, node := range nodes {
		if node != nil && node.NodeType == enum.NodeTypeSection {
			sections = append(sections, node)
		}
	}
	return sections, nil
}

// FindSectionById 按节点ID查询章节，不存在返回 (nil, nil)。
func (a *DocumentAdapter) FindSectionById(ctx context.Context, documentId, nodeId int64) (*entity.StructureNode, error) {
	if documentId == 0 || nodeId == 0 {
		return nil, nil
	}
	sections, err := a.ListSections(ctx, documentId)
	if err != nil {
		return nil, err
	}
	for _, section := range sections {
		if section.ID == nodeId {
			return section, nil
		}
	}
	return nil, nil
}

// FindSectionByCode 按节点编码查询章节，不存在返回 (nil, nil)。
func (a *DocumentAdapter) FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*entity.StructureNode, error) {
	if documentId == 0 || strings.TrimSpace(sectionCode) == "" {
		return nil, nil
	}
	sections, err := a.ListSections(ctx, documentId)
	if err != nil {
		return nil, err
	}
	for _, section := range sections {
		if strings.EqualFold(strings.TrimSpace(section.NodeCode), strings.TrimSpace(sectionCode)) {
			return section, nil
		}
	}
	return nil, nil
}

// CountDocumentsByKbIds 按知识库ID列表统计文档数量。
func (a *DocumentAdapter) CountDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	return a.repo.CountDocumentsByKbIds(ctx, kbIds)
}

// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量。
func (a *DocumentAdapter) CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	return a.repo.CountRetrievableDocumentsByKbIds(ctx, kbIds)
}

// FindDocumentProfileByDocIds 根据文档ID列表查询文档画像。
func (a *DocumentAdapter) FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*entity.DocumentProfile, error) {
	return a.repo.SelectDocumentProfilesByDocIds(ctx, docIds)
}

// FindDocumentProfiles 查询所有文档画像。
func (a *DocumentAdapter) FindDocumentProfiles(ctx context.Context) ([]*entity.DocumentProfile, error) {
	return a.repo.SelectDocumentProfiles(ctx)
}
