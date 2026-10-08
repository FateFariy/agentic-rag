package logic

import (
	"context"
	"errors"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
	errorx "github.com/swiftbit/know-agent/services/knowledge/internal/error"
)

// KnowledgeBaseLogicImpl 知识库管理
type KnowledgeBaseLogicImpl struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
	resolver   *config.Resolver
}

func NewKnowledgeBaseLogicImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, global config.GlobalConfigProvider) *KnowledgeBaseLogicImpl {
	return &KnowledgeBaseLogicImpl{
		repo:       repo,
		docGateway: docGateway,
		resolver:   config.NewResolver(global),
	}
}

// SaveKnowledgeBase 保存/更新知识库
func (k *KnowledgeBaseLogicImpl) SaveKnowledgeBase(ctx context.Context, base *entity.KnowledgeBase) (*entity.KnowledgeBase, error) {
	if strutil.IsBlank(base.BaseName) {
		return nil, errors.New("baseName 不能为空")
	}

	// 名称唯一性校验
	existing, err := k.repo.SelectKnowledgeBaseByBaseName(ctx, strutil.Trim(base.BaseName))
	if err != nil && !errors.Is(err, errorx.ErrKnowledgeBaseNotFound) {
		return nil, err
	}
	if existing.ID != base.ID {
		return nil, errors.New("知识库名称已存在")
	}

	if base.ID > 0 {
		// 更新：查询现有记录确认存在
		_, err = k.repo.SelectKnowledgeBaseById(ctx, base.ID)
		if err != nil {
			return nil, err
		}
		if err = k.repo.UpdateKnowledgeBaseById(ctx, base); err != nil {
			return nil, err
		}
	} else {
		// 新增：分配雪花ID
		base.ID = utils.GetSnowflakeNextID()
		if err = k.repo.InsertKnowledgeBase(ctx, base); err != nil {
			return nil, err
		}
	}

	// 如果标记为默认，清除其他默认标记
	if base.IsDefault == utils.Pointer(1) {
		if err = k.repo.ClearOtherDefaults(ctx, base.ID); err != nil {
			return nil, err
		}
	}

	return base, nil
}

// DeleteKnowledgeBase 删除知识库
func (k *KnowledgeBaseLogicImpl) DeleteKnowledgeBase(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, errors.New("id 不能为空")
	}
	if err := k.repo.DeleteKnowledgeBaseById(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

// ListKnowledgeBases 查询所有知识库列表
func (k *KnowledgeBaseLogicImpl) ListKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	return k.repo.SelectKnowledgeBases(ctx)
}

// GetKnowledgeBase 根据ID查询知识库详情
func (k *KnowledgeBaseLogicImpl) GetKnowledgeBase(ctx context.Context, id int64) (*entity.KnowledgeBase, error) {
	if id <= 0 {
		return nil, errors.New("id 不能为空")
	}
	base, err := k.repo.SelectKnowledgeBaseById(ctx, id)
	if err != nil {
		return nil, err
	}
	return base, nil
}

// UpdateKnowledgeBaseSetting 更新知识库
func (k *KnowledgeBaseLogicImpl) UpdateKnowledgeBaseSetting(ctx context.Context, config *entity.KnowledgeBase) (*entity.KnowledgeBase, error) {
	if config.ID <= 0 {
		return nil, errors.New("id 不能为空")
	}

	// 查询现有记录确认存在
	existing, err := k.repo.SelectKnowledgeBaseById(ctx, config.ID)
	if err != nil {
		return nil, err
	}

	// 仅更新配置JSON字段
	existing.RetrievalConfigJson = config.RetrievalConfigJson
	existing.GraphRagConfigJson = config.GraphRagConfigJson
	existing.MetadataFilterJson = config.MetadataFilterJson

	if err = k.repo.UpdateKnowledgeBaseById(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// ListEnabledKnowledgeBases 查询所有启用的知识库
func (k *KnowledgeBaseLogicImpl) ListEnabledKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	return k.repo.SelectKnowledgeBases(ctx)
}

// ListKnowledgeBasesByIds 根据ID列表查询知识库
func (k *KnowledgeBaseLogicImpl) ListKnowledgeBasesByIds(ctx context.Context, ids []int64) ([]*entity.KnowledgeBase, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return k.repo.SelectKnowledgeBaseByIds(ctx, ids)
}

// GetEnabledKnowledgeBase 根据ID获取知识库，要求记录存在且未软删除，记录不存在时仓储返回 ErrKnowledgeBaseNotFound
func (k *KnowledgeBaseLogicImpl) GetEnabledKnowledgeBase(ctx context.Context, id int64) (*entity.KnowledgeBase, error) {
	if id <= 0 {
		return nil, errors.New("id 不能为空")
	}
	base, err := k.repo.SelectKnowledgeBaseById(ctx, id)
	if err != nil {
		return nil, err
	}
	return base, nil
}

// ResolveIndexingOptions 解析指定知识库的索引构建配置；knowledgeBaseId 为 0 时返回全局默认。
func (k *KnowledgeBaseLogicImpl) ResolveIndexingOptions(ctx context.Context, knowledgeBaseId int64) *vo.IndexingOptions {
	if knowledgeBaseId == 0 {
		return k.resolver.ResolveIndexingOptions(nil)
	}
	base, _ := k.repo.SelectKnowledgeBaseById(ctx, knowledgeBaseId)
	// TODO: resolver 当前恒返回全局默认，查库结果暂未参与 kb 级索引配置合并，待 ResolveIndexingOptions 支持后启用
	return k.resolver.ResolveIndexingOptions([]*entity.KnowledgeBase{base})
}

// GraphTypes 返回知识库配置的实体/关系类型白名单（nil 表示回退全局默认）。
func (k *KnowledgeBaseLogicImpl) GraphTypes(ctx context.Context, knowledgeBaseId int64) ([]string, []string, error) {
	base, err := k.repo.SelectKnowledgeBaseById(ctx, knowledgeBaseId)
	if err != nil {
		return nil, nil, err
	}
	return base.EntityTypes, base.RelationTypes, nil
}

// ListKnowledgeBaseItems 查询知识库列表（包含文档数量和可检索文档数量）
func (k *KnowledgeBaseLogicImpl) ListKnowledgeBaseItems(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	bases, err := k.repo.SelectKnowledgeBases(ctx)
	if err != nil {
		return nil, err
	}
	if len(bases) == 0 {
		return nil, nil
	}

	kbIds := utils.Map(bases, func(base *entity.KnowledgeBase) int64 { return base.ID })
	documentCounts, err := k.docGateway.CountDocumentsByKbIds(ctx, kbIds)
	if err != nil {
		return nil, err
	}
	retrievableCounts, err := k.docGateway.CountRetrievableDocumentsByKbIds(ctx, kbIds)
	if err != nil {
		return nil, err
	}
	for _, base := range bases {
		base.DocumentCount = int(documentCounts[base.ID])
		base.RetrievableDocumentCount = int(retrievableCounts[base.ID])
	}

	return bases, nil
}
