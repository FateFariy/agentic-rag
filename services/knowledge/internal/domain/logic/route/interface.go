package route

import (
	"context"

	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route/rank"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

// KnowledgeRouter 知识路由器
type KnowledgeRouter interface {
	// Route 根据问题进行知识路由
	Route(ctx context.Context, routeCtx *Context) (*vo.KnowledgeRouteDecision, error)

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, routeCtx *Context) error
}

// Ranker 统一排名器
type Ranker interface {
	Rank(ctx context.Context, rankCtx *rank.Context) error
}
