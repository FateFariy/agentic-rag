package main

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/pkg/embedding"
	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	"github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/services/knowledge/api"
	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route/rank"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/persistence"
	portconfig "github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/rpc"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/storage"
	grpcserver "github.com/swiftbit/know-agent/services/knowledge/internal/trigger/grpc"
	"github.com/swiftbit/know-agent/services/knowledge/internal/trigger/handler"
)

// Build 按 knowledge 微服务自有资源完成装配
func Build(c *config.Config) (*infra.Server, error) {
	// 1. knowledge 自有基础设施（仅 MySQL + Embedder + 词法桩 + 本地配置）
	db := infra.NewDB(c.Mysql)
	embedder := embedding.NewOllamaEmbedder(c.Embedding)
	elasticStorage := storage.NewElasticStorage()
	localConfig := portconfig.NewLocalConfig(c)

	// 2. 出站 gRPC 客户端（调用 document 服务）
	documentConn, err := infra.DialGrpc(c.Grpc.DocumentAddr)
	if err != nil {
		return nil, fmt.Errorf("dial document service: %w", err)
	}
	documentClient := rpc.NewKnowledgeDocumentClient(documentv1.NewDocumentServiceClient(documentConn))

	// 3. 仓储
	knowledgeRepo := persistence.NewKnowledgeRepository(db)

	// 4. 领域逻辑
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

	// 5. 应用层 Service
	knowledgeService := handler.NewKnowledgeService(knowledgeLogicImpl, knowledgeBaseLogicImpl)

	// 6. gRPC server 注册回调（同时供聚合 Server 与 app 包测试入口复用）
	registerGRPC := func(g *grpc.Server) {
		knowledgev1.RegisterKnowledgeServiceServer(g, grpcserver.NewKnowledgeGrpcServer(retrievalScopeLogicImpl, knowledgeRouter, knowledgeBaseLogicImpl))
	}

	// 7. 装配聚合 HTTP + gRPC Server
	httpServer := infra.NewHTTPServer(c.Http, func(s *rest.Server) { api.RegisterHandlers(s, knowledgeService) })
	rpcServer := infra.NewRPCServer(c.Rpc, registerGRPC)

	return infra.NewServer(httpServer, rpcServer), nil
}
