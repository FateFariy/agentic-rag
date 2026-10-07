// Package bootstrap 为 knowledge 微服务提供进程内装配
package main

import (
	"google.golang.org/grpc"

	"github.com/swiftbit/know-agent/pkg/embedding"
	"github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route/rank"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/persistence"
	portconfig "github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/services/knowledge/internal/server"
	grpcserver "github.com/swiftbit/know-agent/services/knowledge/internal/trigger/grpc"
	"github.com/swiftbit/know-agent/services/knowledge/internal/trigger/handler"
)

// Build 按 knowledge 微服务自有资源完成装配。documentClient 为调用 document
// 服务的 gRPC 客户端，由 cmd 或公开装配入口（services/knowledge/app）注入。
func Build(c *config.Config, documentClient *gateway.KnowledgeDocumentClient) *server.Server {
	// 1. knowledge 自有基础设施（仅 MySQL + Embedder + 词法桩 + 本地配置）
	db := infra.NewDB(c.Mysql)
	embedder := embedding.NewOllamaEmbedder(c.Embedding)
	elasticStorage := storage.NewElasticStorage()
	localConfig := portconfig.NewLocalConfig(c)

	// 2. 仓储
	knowledgeRepo := persistence.NewKnowledgeRepository(db)

	// 3. 领域逻辑
	retrievalScopeLogicImpl := logic.NewKnowledgeBaseRetrievalScopeLogicImpl(knowledgeRepo, documentClient, localConfig)
	opts := []rank.Option{rank.WithLexicalIndex(elasticStorage), rank.WithEmbedding(embedder)}
	rankers := []route.Ranker{
		rank.NewDocumentRanker(knowledgeRepo, documentClient, opts...),
		rank.NewScopeRanker(knowledgeRepo, documentClient, opts...),
		rank.NewTopicRanker(knowledgeRepo, documentClient, opts...),
	}
	knowledgeRouter := route.NewKnowledgeRouteImpl(knowledgeRepo, documentClient, rankers, route.WithEmbedding(embedder))
	knowledgeLogicImpl := logic.NewKnowledgeLogicImpl(knowledgeRepo, documentClient)
	knowledgeBaseLogicImpl := logic.NewKnowledgeBaseLogicImpl(knowledgeRepo, documentClient, localConfig)

	// 4. 应用层 Service
	knowledgeService := handler.NewKnowledgeService(knowledgeLogicImpl, knowledgeBaseLogicImpl)

	// 5. gRPC server 注册回调（同时供聚合 Server 与 app 包测试入口复用）
	registerGRPC := func(g *grpc.Server) {
		knowledgev1.RegisterKnowledgeServiceServer(g, grpcserver.NewKnowledgeGrpcServer(retrievalScopeLogicImpl, knowledgeRouter, knowledgeBaseLogicImpl))
	}

	// 6. 装配聚合 HTTP + gRPC Server
	httpServer := server.NewHTTPServer(c.Http, knowledgeService)
	rpcServer := server.NewRPCServer(c.Grpc.KnowledgeRpc, registerGRPC)

	return server.NewServer(httpServer, rpcServer)
}
