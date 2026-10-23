package logic

import (
	"context"
	"mime/multipart"

	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

// LifecycleLogic 生命周期逻辑接口
type LifecycleLogic interface {
	// Upload 上传文档
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, document *entity.Document) (*entity.Document, int64, error)

	// QueryDocumentPage 分页查询文档列表
	QueryDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error)

	// QueryDocumentDetail 查询文档详情
	QueryDocumentDetail(ctx context.Context, documentId int64) (*entity.Document, error)

	// DeleteDocument 删除文档
	DeleteDocument(ctx context.Context, documentId int64) (string, error)

	// QueryStrategyPlan 查询策略方案
	QueryStrategyPlan(ctx context.Context, documentId int64) (*entity.Document, *entity.DocumentStrategyPlan, error)

	// ConfirmStrategy 确认策略
	ConfirmStrategy(ctx context.Context, cmd *vo.DocumentStrategyConfirmCmd) (*entity.DocumentStrategyPlan, *entity.Document, error)

	// BuildIndex 构建索引
	BuildIndex(ctx context.Context, documentId, planId, operatorId int64) (*vo.DocumentIndexBuild, error)

	// QueryDocumentChunks 查询文档块
	QueryDocumentChunks(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, int64, error)

	// QueryDocumentChunkDetail 查询文档块详情
	QueryDocumentChunkDetail(ctx context.Context, documentId, taskId, chunkId int64) (*entity.DocumentChunkDetail, error)

	// QueryTaskLogs 查询任务日志
	QueryTaskLogs(ctx context.Context, taskId int64, pageNo, pageSize int) (*entity.DocumentTask, int64, error)

	// ListRetrievableDocuments 获取可检索的文档列表
	ListRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.DocumentMetadata, error)
}

// DocumentQueryLogic document 服务对外检索只读用例
type DocumentQueryLogic interface {
	// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据
	FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error)

	// FindRetrieveDocumentByIds 根据文档ID列表查询可检索的文档元数据
	FindRetrieveDocumentByIds(ctx context.Context, ids []int64) ([]*vo.DocumentMetadata, error)

	// FindParentChunks 根据父块ID列表查询父块
	FindParentChunks(ctx context.Context, ids []int64) ([]*entity.DocumentParentChunk, error)

	// ListSections 查询指定文档的章节结构节点（仅保留章节类型节点）
	ListSections(ctx context.Context, documentId int64) ([]*entity.StructureNode, error)

	// FindSectionById 按节点ID查询章节，不存在返回 (nil, nil)
	FindSectionById(ctx context.Context, documentId, nodeId int64) (*entity.StructureNode, error)

	// FindSectionByCode 按节点编码查询章节，不存在返回 (nil, nil)
	FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*entity.StructureNode, error)

	// CountDocumentsByKbIds 按知识库ID列表统计文档数量
	CountDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error)

	// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量
	CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error)
}
