package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/services/knowledge/internal/convert"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	errorx "github.com/swiftbit/know-agent/services/knowledge/internal/error"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/persistence/model"
)

// KnowledgeRepositoryImpl 文档知识仓储实现
type KnowledgeRepositoryImpl struct {
	*infra.TxManager
}

var _ adapter.KnowledgeRepository = (*KnowledgeRepositoryImpl)(nil)

// NewKnowledgeRepository 构造函数，仅依赖 knowledge 服务自有的 MySQL 连接
func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepositoryImpl {
	return &KnowledgeRepositoryImpl{
		TxManager: infra.NewTxManager(db),
	}
}

// ============ 知识范围节点 ============

// SelectScopes 查询知识范围节点列表，kbIds 为可选参数：不传时查询全部，否则按知识库ID过滤
func (k *KnowledgeRepositoryImpl) SelectScopes(ctx context.Context, kbIds ...int64) ([]*entity.KnowledgeScopeNode, error) {
	var nodes []*entity.KnowledgeScopeNode
	builder := k.DBWithContext(ctx).Model(&model.KnowledgeScopeNode{}).
		Order("sort_order ASC, id ASC")
	if len(kbIds) > 0 {
		builder = builder.Where("knowledge_base_id IN ?", kbIds)
	}
	if err := builder.Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// SelectScopeById 根据ID查询知识范围节点
func (k *KnowledgeRepositoryImpl) SelectScopeById(ctx context.Context, id int64, kbId int64) (*entity.KnowledgeScopeNode, error) {
	var scope entity.KnowledgeScopeNode
	builder := k.DBWithContext(ctx).Model(&model.KnowledgeScopeNode{}).Where("id = ?", id)
	if kbId > 0 {
		builder = builder.Where("knowledge_base_id = ?", kbId)
	}
	if err := builder.First(&scope).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &scope, nil
}

// SelectScopeByName 按名称查询知识范围（用于唯一性校验，excludeId 为排除的ID）
func (k *KnowledgeRepositoryImpl) SelectScopeByName(ctx context.Context, kbId int64, scopeName string, excludeId int64) (*entity.KnowledgeScopeNode, error) {
	var scope entity.KnowledgeScopeNode
	builder := k.DBWithContext(ctx).Model(&model.KnowledgeScopeNode{}).
		Where("knowledge_base_id = ? AND scope_name = ?", kbId, scopeName)
	if excludeId > 0 {
		builder = builder.Where("id != ?", excludeId)
	}
	if err := builder.First(&scope).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &scope, nil
}

// CountChildScopes 统计子级知识范围数量
func (k *KnowledgeRepositoryImpl) CountChildScopes(ctx context.Context, kbId int64, parentId int64) (int64, error) {
	var count int64
	err := k.DBWithContext(ctx).Model(&model.KnowledgeScopeNode{}).
		Where("knowledge_base_id = ? AND parent_scope_id = ?", kbId, parentId).
		Count(&count).Error
	return count, err
}

// CountTopicsByScope 统计范围下的主题数量
func (k *KnowledgeRepositoryImpl) CountTopicsByScope(ctx context.Context, kbId int64, scopeId int64) (int64, error) {
	var count int64
	err := k.DBWithContext(ctx).Model(&model.KnowledgeTopicNode{}).
		Where("knowledge_base_id = ? AND scope_id = ?", kbId, scopeId).
		Count(&count).Error
	return count, err
}

// UpsertKnowledgeScopeNode 插入或更新知识范围节点
func (k *KnowledgeRepositoryImpl) UpsertKnowledgeScopeNode(ctx context.Context, node *entity.KnowledgeScopeNode) error {
	nodeModel := convert.ToKnowledgeScopeNodeModel(node)
	if err := k.DBWithContext(ctx).Model(&model.KnowledgeScopeNode{}).Where("id = ?", node.ID).First(node).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return k.DBWithContext(ctx).Create(nodeModel).Error
	}
	nodeModel.ID = node.ID
	return k.DBWithContext(ctx).Updates(nodeModel).Error
}

// DeleteScope 删除知识范围
func (k *KnowledgeRepositoryImpl) DeleteScope(ctx context.Context, id int64, kbId int64) error {
	return k.DBWithContext(ctx).Where("id = ? AND knowledge_base_id = ?", id, kbId).
		Delete(&model.KnowledgeScopeNode{}).Error
}

// ============ 主题节点 ============

// SelectKnowledgeTopicNodesByKbIds 根据知识库ID获取有效的主题节点
func (k *KnowledgeRepositoryImpl) SelectKnowledgeTopicNodesByKbIds(ctx context.Context, kbIds []int64) ([]*entity.KnowledgeTopicNode, error) {
	var nodes []*entity.KnowledgeTopicNode
	if err := k.DBWithContext(ctx).Model(&model.KnowledgeTopicNode{}).
		Where("knowledge_base_id IN ?", kbIds).
		Order("sort_order ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// ListTopics 按知识库ID和范围ID查询主题列表
func (k *KnowledgeRepositoryImpl) ListTopics(ctx context.Context, kbId int64, scopeId int64) ([]*entity.KnowledgeTopicNode, error) {
	var nodes []*entity.KnowledgeTopicNode
	builder := k.DBWithContext(ctx).Model(&model.KnowledgeTopicNode{}).
		Order("sort_order ASC, id ASC")
	if kbId > 0 {
		builder = builder.Where("knowledge_base_id = ?", kbId)
	}
	if scopeId > 0 {
		builder = builder.Where("scope_id = ?", scopeId)
	}
	if err := builder.Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// SelectTopicByID 根据ID查询主题节点
func (k *KnowledgeRepositoryImpl) SelectTopicByID(ctx context.Context, id int64, kbId int64) (*entity.KnowledgeTopicNode, error) {
	var topic entity.KnowledgeTopicNode
	builder := k.DBWithContext(ctx).Model(&model.KnowledgeTopicNode{}).Where("id = ?", id)
	if kbId > 0 {
		builder = builder.Where("knowledge_base_id = ?", kbId)
	}
	if err := builder.First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &topic, nil
}

// SelectTopicByName 按名称查询主题（用于唯一性校验，excludeId 为排除的ID）
func (k *KnowledgeRepositoryImpl) SelectTopicByName(ctx context.Context, kbId int64, scopeId int64, topicName string, excludeId int64) (*entity.KnowledgeTopicNode, error) {
	var topic entity.KnowledgeTopicNode
	builder := k.DBWithContext(ctx).Model(&model.KnowledgeTopicNode{}).
		Where("knowledge_base_id = ? AND scope_id = ? AND topic_name = ?", kbId, scopeId, topicName)
	if excludeId > 0 {
		builder = builder.Where("id != ?", excludeId)
	}
	if err := builder.First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &topic, nil
}

// CountRelationsByTopic 统计主题下的文档关联数量
func (k *KnowledgeRepositoryImpl) CountRelationsByTopic(ctx context.Context, kbId int64, topicId int64) (int64, error) {
	var count int64
	err := k.DBWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Where("knowledge_base_id = ? AND topic_id = ?", kbId, topicId).
		Count(&count).Error
	return count, err
}

// UpsertKnowledgeTopicNode 插入或更新主题节点
func (k *KnowledgeRepositoryImpl) UpsertKnowledgeTopicNode(ctx context.Context, node *entity.KnowledgeTopicNode) error {
	nodeModel := convert.ToKnowledgeTopicNodeModel(node)
	if err := k.DBWithContext(ctx).Model(&model.KnowledgeTopicNode{}).
		Where("id = ?", node.ID).
		First(node).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return k.DBWithContext(ctx).Create(nodeModel).Error
	}
	nodeModel.ID = node.ID
	return k.DBWithContext(ctx).Updates(nodeModel).Error
}

// DeleteTopic 软删除主题（GORM 软删除）
func (k *KnowledgeRepositoryImpl) DeleteTopic(ctx context.Context, id int64, kbId int64) error {
	return k.DBWithContext(ctx).Where("id = ? AND knowledge_base_id = ?", id, kbId).
		Delete(&model.KnowledgeTopicNode{}).Error
}

// ============ 主题-文档关系 ============

// SelectTopicDocumentRelationsByKbIds 按知识库ID列表批量查询全部有效的主题-文档关联（供检索范围收窄加载 topic 绑定）
func (k *KnowledgeRepositoryImpl) SelectTopicDocumentRelationsByKbIds(ctx context.Context, kbIds []int64) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	if len(kbIds) == 0 {
		return []*entity.KnowledgeTopicDocumentRelation{}, nil
	}
	var relations []*entity.KnowledgeTopicDocumentRelation
	if err := k.DBWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Where("knowledge_base_id IN ?", kbIds).
		Order("topic_id ASC, relation_score DESC, id ASC").
		Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// ListTopicDocumentRelations 按知识库ID和主题ID查询主题-文档关系列表
func (k *KnowledgeRepositoryImpl) ListTopicDocumentRelations(ctx context.Context, kbId int64, topicId int64) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	var relations []*entity.KnowledgeTopicDocumentRelation
	builder := k.DBWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Order("relation_score DESC, id DESC")
	if kbId > 0 {
		builder = builder.Where("knowledge_base_id = ?", kbId)
	}
	if topicId > 0 {
		builder = builder.Where("topic_id = ?", topicId)
	}
	if err := builder.Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// SelectTopicDocumentRelation 查询已存在的主题-文档关系
func (k *KnowledgeRepositoryImpl) SelectTopicDocumentRelation(ctx context.Context, kbId int64, topicId int64, documentId int64) (*entity.KnowledgeTopicDocumentRelation, error) {
	var relation entity.KnowledgeTopicDocumentRelation
	err := k.DBWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Where("knowledge_base_id = ? AND topic_id = ? AND document_id = ?", kbId, topicId, documentId).
		First(&relation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &relation, nil
}

// UpsertTopicDocumentRelation 插入或更新主题-文档关系
func (k *KnowledgeRepositoryImpl) UpsertTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) error {
	relModel := convert.ToTopicDocumentRelationModel(relation)
	if err := k.DBWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Where("id = ?", relation.ID).
		First(relation).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return k.DBWithContext(ctx).Create(relModel).Error
	}
	relModel.ID = relation.ID
	return k.DBWithContext(ctx).Updates(relModel).Error
}

// DeleteTopicDocumentRelation 软删除主题-文档关联
func (k *KnowledgeRepositoryImpl) DeleteTopicDocumentRelation(ctx context.Context, kbId int64, topicId int64, documentId int64) (int64, error) {
	res := k.DBWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Where("knowledge_base_id = ? AND topic_id = ? AND document_id = ?", kbId, topicId, documentId).
		Delete(nil)
	return res.RowsAffected, res.Error
}

// ========== 知识库相关 ==========

// InsertKnowledgeBase 插入知识库
func (k *KnowledgeRepositoryImpl) InsertKnowledgeBase(ctx context.Context, base *entity.KnowledgeBase) error {
	return k.DBWithContext(ctx).Create(convert.ToKnowledgeBaseModel(base)).Error
}

// UpdateKnowledgeBaseById 根据ID更新知识库
func (k *KnowledgeRepositoryImpl) UpdateKnowledgeBaseById(ctx context.Context, base *entity.KnowledgeBase) error {
	return k.DBWithContext(ctx).Model(&model.KnowledgeBase{}).
		Where("id = ?", base.ID).
		Updates(convert.ToKnowledgeBaseModel(base)).Error
}

// DeleteKnowledgeBaseById 根据ID删除知识库
func (k *KnowledgeRepositoryImpl) DeleteKnowledgeBaseById(ctx context.Context, id int64) error {
	return k.DBWithContext(ctx).Delete(&model.KnowledgeBase{}, id).Error
}

// SelectKnowledgeBaseById 根据ID查询知识库
func (k *KnowledgeRepositoryImpl) SelectKnowledgeBaseById(ctx context.Context, id int64) (*entity.KnowledgeBase, error) {
	var base *model.KnowledgeBase
	if err := k.DBWithContext(ctx).Where("id = ?", id).First(&base).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrKnowledgeBaseNotFound.Format(id)
		}
		return nil, err
	}
	return convert.ToKnowledgeBaseEntity(base), nil
}

// SelectKnowledgeBases 查询所有知识库
func (k *KnowledgeRepositoryImpl) SelectKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	var bases []*model.KnowledgeBase
	if err := k.DBWithContext(ctx).
		Order("sort_order ASC, id ASC").
		Find(&bases).Error; err != nil {
		return nil, err
	}
	return convert.ToKnowledgeBaseEntities(bases), nil
}

// SelectKnowledgeBaseByIds 根据ID列表查询知识库
func (k *KnowledgeRepositoryImpl) SelectKnowledgeBaseByIds(ctx context.Context, ids []int64) ([]*entity.KnowledgeBase, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var bases []*model.KnowledgeBase
	if err := k.DBWithContext(ctx).
		Where("id IN ?", ids).
		Order("sort_order ASC, id ASC").
		Find(&bases).Error; err != nil {
		return nil, err
	}
	return convert.ToKnowledgeBaseEntities(bases), nil
}

// SelectKnowledgeBaseByBaseName 根据名称查询知识库
func (k *KnowledgeRepositoryImpl) SelectKnowledgeBaseByBaseName(ctx context.Context, baseName string) (*entity.KnowledgeBase, error) {
	var base entity.KnowledgeBase
	if err := k.DBWithContext(ctx).Model(&model.KnowledgeBase{}).
		Where("base_name = ?", baseName).
		First(&base).Error; err != nil {
		return nil, err
	}
	return &base, nil
}

// ClearOtherDefaults 清除其他知识库的默认标记
func (k *KnowledgeRepositoryImpl) ClearOtherDefaults(ctx context.Context, currentId int64) error {
	return k.DBWithContext(ctx).Model(&model.KnowledgeBase{}).
		Where("id != ? AND is_default = ?", currentId, 1).
		Update("is_default", 0).Error
}
