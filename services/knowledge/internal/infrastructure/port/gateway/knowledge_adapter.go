package gateway

import (
	"context"

	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/aggregate"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

// KnowledgeAdapter 是 knowledge 微服务面向 gRPC 入站的应用编排层
type KnowledgeAdapter struct {
	l        logic.KnowledgeBaseRetrievalLogic
	repo     adapter.KnowledgeRepository
	resolver *config.Resolver
	router   route.KnowledgeRouter
}

// NewKnowledgeAdapter 创建知识库解析器
func NewKnowledgeAdapter(l logic.KnowledgeBaseRetrievalLogic, repo adapter.KnowledgeRepository, router route.KnowledgeRouter, global config.GlobalConfigProvider) *KnowledgeAdapter {
	return &KnowledgeAdapter{
		l:        l,
		repo:     repo,
		router:   router,
		resolver: config.NewResolver(global),
	}
}

// DetermineKnowledgeScope 根据知识库选择模式解析检索范围，返回 knowledge 自有快照。
func (r *KnowledgeAdapter) DetermineKnowledgeScope(ctx context.Context, selectMode string, kbIds []string) (*aggregate.KnowledgeBaseSelectionSnapshot, error) {
	return r.l.DetermineKnowledgeScope(ctx, selectMode, kbIds)
}

// Route 根据路由上下文执行知识路由，返回 knowledge 自有路由决策。
func (r *KnowledgeAdapter) Route(ctx context.Context, routeCtx *route.Context) (*vo.KnowledgeRouteDecision, error) {
	return r.router.Route(ctx, routeCtx)
}

// RecordShadowRoute 记录影子路由结果。
func (r *KnowledgeAdapter) RecordShadowRoute(ctx context.Context, routeCtx *route.Context) error {
	return r.router.RecordShadowRoute(ctx, routeCtx)
}

// Resolve 解析指定知识库的索引构建配置；knowledgeBaseId 为 0 时返回全局默认。
func (r *KnowledgeAdapter) Resolve(ctx context.Context, knowledgeBaseId int64) *vo.IndexingOptions {
	if knowledgeBaseId == 0 {
		return r.resolver.ResolveIndexingOptions(nil)
	}
	base, _ := r.repo.SelectKnowledgeBaseById(ctx, knowledgeBaseId)
	return r.resolver.ResolveIndexingOptions([]*entity.KnowledgeBase{base})
}

// RequireEnabled 校验知识库可用并返回其实体。
func (r *KnowledgeAdapter) RequireEnabled(ctx context.Context, knowledgeBaseId int64) (*entity.KnowledgeBase, error) {
	return r.repo.SelectKnowledgeBaseById(ctx, knowledgeBaseId)
}

// GraphTypes 返回知识库配置的实体/关系类型白名单（nil 表示回退全局默认）。
func (r *KnowledgeAdapter) GraphTypes(ctx context.Context, knowledgeBaseId int64) ([]string, []string, error) {
	base, err := r.repo.SelectKnowledgeBaseById(ctx, knowledgeBaseId)
	if err != nil {
		return nil, nil, err
	}
	return base.EntityTypes, base.RelationTypes, nil
}
