package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
)

// KnowledgeRepository 领域模型视角下的知识库仓储：提供可检索文档、父级块、知识范围/主题节点等能力
type KnowledgeRepository interface {
	// Do 运行一个事务
	Do(ctx context.Context, fn func(ctx context.Context) error) error

	// ========== 知识范围节点相关 ==========

	// SelectScopes 查询知识范围节点列表，kbIds 为可选参数：不传（或全为非正数）时查询全部，否则按知识库ID过滤
	SelectScopes(ctx context.Context, kbIds ...int64) ([]*entity.KnowledgeScopeNode, error)

	// SelectScopeById 根据ID查询知识范围节点
	SelectScopeById(ctx context.Context, id int64, kbId int64) (*entity.KnowledgeScopeNode, error)

	// SelectScopeByName 按名称查询知识范围（用于唯一性校验，excludeId 为排除的ID）
	SelectScopeByName(ctx context.Context, kbId int64, scopeName string, excludeId int64) (*entity.KnowledgeScopeNode, error)

	// CountChildScopes 统计子级知识范围数量
	CountChildScopes(ctx context.Context, kbId int64, parentId int64) (int64, error)

	// CountTopicsByScope 统计范围下的主题数量
	CountTopicsByScope(ctx context.Context, kbId int64, scopeId int64) (int64, error)

	// UpsertKnowledgeScopeNode 保存/更新知识范围节点
	UpsertKnowledgeScopeNode(ctx context.Context, node *entity.KnowledgeScopeNode) error

	// DeleteScope 删除知识范围
	DeleteScope(ctx context.Context, id int64, kbId int64) error

	// ========== 主题节点相关 ==========

	// SelectKnowledgeTopicNodesByKbIds 根据知识库ID获取有效的主题节点
	SelectKnowledgeTopicNodesByKbIds(ctx context.Context, kbIds []int64) ([]*entity.KnowledgeTopicNode, error)

	// ListTopics 按知识库ID和范围ID查询主题列表
	ListTopics(ctx context.Context, kbId int64, scopeId int64) ([]*entity.KnowledgeTopicNode, error)

	// SelectTopicByID 根据ID查询主题节点
	SelectTopicByID(ctx context.Context, id int64, kbId int64) (*entity.KnowledgeTopicNode, error)

	// SelectTopicByName 按名称查询主题（用于唯一性校验，excludeId 为排除的ID）
	SelectTopicByName(ctx context.Context, kbId int64, scopeId int64, topicName string, excludeId int64) (*entity.KnowledgeTopicNode, error)

	// CountRelationsByTopic 统计主题下的文档关联数量
	CountRelationsByTopic(ctx context.Context, kbId int64, topicId int64) (int64, error)

	// UpsertKnowledgeTopicNode 保存/更新主题节点
	UpsertKnowledgeTopicNode(ctx context.Context, node *entity.KnowledgeTopicNode) error

	// DeleteTopic 软删除主题（设置状态为无效）
	DeleteTopic(ctx context.Context, id int64, kbId int64) error

	// ========== 主题-文档关系相关 ==========

	// SelectTopicDocumentRelationsByKbIds 按知识库ID列表批量查询全部有效的主题-文档关联（供检索范围收窄加载 topic 绑定）
	SelectTopicDocumentRelationsByKbIds(ctx context.Context, kbIds []int64) ([]*entity.KnowledgeTopicDocumentRelation, error)

	// ListTopicDocumentRelations 按知识库ID和主题ID查询主题-文档关系列表
	ListTopicDocumentRelations(ctx context.Context, kbId int64, topicId int64) ([]*entity.KnowledgeTopicDocumentRelation, error)

	// SelectTopicDocumentRelation 查询已存在的主题-文档关系
	SelectTopicDocumentRelation(ctx context.Context, kbId int64, topicId int64, documentId int64) (*entity.KnowledgeTopicDocumentRelation, error)

	// UpsertTopicDocumentRelation 保存/更新主题-文档关联
	UpsertTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) error

	// DeleteTopicDocumentRelation 软删除主题-文档关联
	DeleteTopicDocumentRelation(ctx context.Context, kbId int64, topicId int64, documentId int64) (int64, error)

	// ========== 知识库配置相关 ==========

	// InsertKnowledgeBase 插入知识库
	InsertKnowledgeBase(ctx context.Context, config *entity.KnowledgeBase) error

	// UpdateKnowledgeBaseById 根据ID更新知识库
	UpdateKnowledgeBaseById(ctx context.Context, config *entity.KnowledgeBase) error

	// DeleteKnowledgeBaseById 根据ID删除知识库
	DeleteKnowledgeBaseById(ctx context.Context, id int64) error

	// SelectKnowledgeBaseById 根据ID查询知识库
	SelectKnowledgeBaseById(ctx context.Context, id int64) (*entity.KnowledgeBase, error)

	// SelectKnowledgeBases 查询所有知识库
	SelectKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error)

	// SelectKnowledgeBaseByIds 根据ID列表查询知识库
	SelectKnowledgeBaseByIds(ctx context.Context, ids []int64) ([]*entity.KnowledgeBase, error)

	// SelectKnowledgeBaseByBaseName 根据名称查询知识库
	SelectKnowledgeBaseByBaseName(ctx context.Context, baseName string) (*entity.KnowledgeBase, error)

	// ClearOtherDefaults 清除其他知识库的默认标记
	ClearOtherDefaults(ctx context.Context, currentId int64) error
}
