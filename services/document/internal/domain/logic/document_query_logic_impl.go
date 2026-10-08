package logic

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/services/document/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// 编译期断言：DocumentQueryLogicImpl 实现 DocumentQueryLogic
var _ DocumentQueryLogic = (*DocumentQueryLogicImpl)(nil)

// DocumentQueryLogicImpl 实现 DocumentQueryLogic，仅依赖 DocumentRepository
type DocumentQueryLogicImpl struct {
	repo adapter.DocumentRepository
}

// NewDocumentQueryLogicImpl 创建 document 对外查询领域逻辑
func NewDocumentQueryLogicImpl(repo adapter.DocumentRepository) *DocumentQueryLogicImpl {
	return &DocumentQueryLogicImpl{repo: repo}
}

// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据
func (l *DocumentQueryLogicImpl) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error) {
	return l.repo.SelectRetrievableDocumentsByKbIds(ctx, kbIds...)
}

// FindRetrieveDocumentByIds 根据文档ID列表查询可检索的文档元数据
func (l *DocumentQueryLogicImpl) FindRetrieveDocumentByIds(ctx context.Context, ids []int64) ([]*vo.DocumentMetadata, error) {
	return l.repo.SelectRetrievableDocumentsByIds(ctx, ids...)
}

// FindParentChunks 根据父块ID列表查询父块
func (l *DocumentQueryLogicImpl) FindParentChunks(ctx context.Context, ids []int64) ([]*entity.DocumentParentChunk, error) {
	return l.repo.SelectParentChunkListByIds(ctx, ids)
}

// ListSections 查询指定文档的章节结构节点（仅保留章节类型节点）
func (l *DocumentQueryLogicImpl) ListSections(ctx context.Context, documentId int64) ([]*entity.StructureNode, error) {
	nodes, err := l.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	return entity.StructureNodes(nodes).Sections(), nil
}

// FindSectionById 按节点ID查询章节，不存在返回 (nil, nil)
func (l *DocumentQueryLogicImpl) FindSectionById(ctx context.Context, documentId, nodeId int64) (*entity.StructureNode, error) {
	if documentId == 0 || nodeId == 0 {
		return nil, nil
	}
	nodes, err := l.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	return entity.StructureNodes(nodes).Sections().FindSectionByID(nodeId), nil
}

// FindSectionByCode 按节点编码查询章节，不存在返回 (nil, nil)
func (l *DocumentQueryLogicImpl) FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*entity.StructureNode, error) {
	if documentId == 0 || strings.TrimSpace(sectionCode) == "" {
		return nil, nil
	}
	nodes, err := l.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	return entity.StructureNodes(nodes).Sections().FindSectionByCode(sectionCode), nil
}

// CountDocumentsByKbIds 按知识库ID列表统计文档数量
func (l *DocumentQueryLogicImpl) CountDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	return l.repo.CountDocumentsByKbIds(ctx, kbIds)
}

// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量
func (l *DocumentQueryLogicImpl) CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	return l.repo.CountRetrievableDocumentsByKbIds(ctx, kbIds)
}

// FindDocumentProfileByDocIds 根据文档ID列表查询文档画像
func (l *DocumentQueryLogicImpl) FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*entity.DocumentProfile, error) {
	return l.repo.SelectDocumentProfilesByDocIds(ctx, docIds)
}

// FindDocumentProfiles 查询所有文档画像
func (l *DocumentQueryLogicImpl) FindDocumentProfiles(ctx context.Context) ([]*entity.DocumentProfile, error) {
	return l.repo.SelectDocumentProfiles(ctx)
}
