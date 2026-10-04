// Package bootstrap 为 knowledge 微服务提供进程内装配
package bootstrap

import (
	"github.com/zeromicro/go-zero/rest"
	"google.golang.org/grpc"

	"github.com/swiftbit/know-agent/pkg/config"
	"github.com/swiftbit/know-agent/pkg/emb"
	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/proto/knowagent/knowledge/v1"
	"github.com/swiftbit/know-agent/pkg/httpserver"
	"github.com/swiftbit/know-agent/pkg/mysql"
	"github.com/swiftbit/know-agent/services/knowledge/api"
	knowlogic "github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic"
	knowroute "github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route/rank"
	grpcserver "github.com/swiftbit/know-agent/services/knowledge/internal/grpc/server"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/persistence"
	portconfig "github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/services/knowledge/internal/trigger/handler"
)

// App 承载 knowledge 服务启动所需的 HTTP 服务与 gRPC 注册回调。
type App struct {
	httpServer   *rest.Server
	registerGRPC func(*grpc.Server)
}

// HTTPServer 返回仅注册 knowledge 路由的 HTTP 服务。
func (a *App) HTTPServer() *rest.Server {
	return a.httpServer
}

// RegisterGRPC 向指定 gRPC server 注册 knowledge 服务。
func (a *App) RegisterGRPC(s *grpc.Server) {
	a.registerGRPC(s)
}

// Build 按 knowledge 微服务自有资源完成装配。documentForKnowledge 为调用 document
// 服务的 gRPC 客户端，由 cmd 或公开装配入口（services/knowledge/app）注入。
func Build(c *config.Config, documentForKnowledge *gateway.KnowledgeDocumentClient) *App {
	// 1. knowledge 自有基础设施（仅 MySQL + Embedding + 词法桩 + 本地配置）
	db := mysql.NewDB(c.Mysql)
	embedder := emb.NewOllamaEmbedder(c.Embedding)
	elasticStorage := storage.NewElasticStorage()
	localConfig := portconfig.NewLocalConfig(c)

	// 2. 仓储
	knowledgeRepo := persistence.NewKnowledgeRepository(db)

	// 3. 领域逻辑
	retrievalScopeLogicImpl := knowlogic.NewKnowledgeBaseRetrievalScopeLogicImpl(knowledgeRepo, documentForKnowledge, localConfig)
	opts := []rank.Option{rank.WithLexicalIndex(elasticStorage), rank.WithEmbedding(embedder)}
	rankers := []knowroute.Ranker{
		rank.NewDocumentRanker(knowledgeRepo, documentForKnowledge, opts...),
		rank.NewScopeRanker(knowledgeRepo, documentForKnowledge, opts...),
		rank.NewTopicRanker(knowledgeRepo, documentForKnowledge, opts...),
	}
	knowledgeRouter := knowroute.NewKnowledgeRouteImpl(knowledgeRepo, documentForKnowledge, rankers, knowroute.WithEmbedding(embedder))
	knowledgeLogicImpl := knowlogic.NewKnowledgeLogicImpl(knowledgeRepo, documentForKnowledge)
	knowledgeBaseLogicImpl := knowlogic.NewKnowledgeBaseLogicImpl(knowledgeRepo, documentForKnowledge)
	knowledgeAdapter := gateway.NewKnowledgeAdapter(retrievalScopeLogicImpl, knowledgeRepo, knowledgeRouter, localConfig)

	// 4. 应用层 Service
	knowledgeService := handler.NewKnowledgeService(knowledgeLogicImpl, knowledgeBaseLogicImpl)

	// 5. 仅注册 knowledge 自己的 HTTP 路由
	httpServer := httpserver.NewRestServer(c.Http)
	api.RegisterHandlers(httpServer, knowledgeService)

	// 6. gRPC server 注册回调
	registerGRPC := func(g *grpc.Server) {
		knowledgev1.RegisterKnowledgeServiceServer(g, grpcserver.NewKnowledgeServer(knowledgeAdapter))
	}
	return &App{httpServer: httpServer, registerGRPC: registerGRPC}
}
