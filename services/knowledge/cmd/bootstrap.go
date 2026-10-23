package main

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/pkg/ai"
	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	"github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/services/knowledge/api"
	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/scope"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/persistence"
	portconfig "github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/rpc"
	grpcserver "github.com/swiftbit/know-agent/services/knowledge/internal/trigger/grpc"
	"github.com/swiftbit/know-agent/services/knowledge/internal/trigger/handler"
)

// Build 按 knowledge 微服务自有资源完成装配
func Build(c *config.Config) (*infra.Server, error) {
	// 1. knowledge 自有基础设施（仅 MySQL + 本地配置）
	db := infra.NewDB(c.Mysql)
	localConfig := portconfig.NewLocalConfig(c)

	// 2. 出站 gRPC 客户端（调用 document 服务）
	documentConn, err := infra.DialGrpc(c.Grpc.DocumentAddr)
	if err != nil {
		return nil, fmt.Errorf("dial document service: %w", err)
	}
	documentClient := rpc.NewKnowledgeDocumentClient(documentv1.NewDocumentServiceClient(documentConn))

	// 3. 仓储
	knowledgeRepo := persistence.NewKnowledgeRepository(db)

	// 3.1 共享本地 LLM 网关客户端（与 chat query_rewrite 同款 Ollama 接入，无云端依赖）+ 意图打分器
	judgeModel := ai.NewOllamaModel(c.JudgeModel)
	intentScorer := scope.NewKnowledgeIntentScorer(knowledgeRepo, judgeModel, ai.NewRendererImpl(), 0)

	// 4. 领域逻辑
	retrievalScopeLogicImpl := logic.NewKnowledgeBaseRetrievalScopeLogicImpl(knowledgeRepo, documentClient, localConfig, intentScorer)
	knowledgeLogicImpl := logic.NewKnowledgeLogicImpl(knowledgeRepo, documentClient)
	knowledgeBaseLogicImpl := logic.NewKnowledgeBaseLogicImpl(knowledgeRepo, documentClient, localConfig)

	// 5. 应用层 Service
	knowledgeService := handler.NewKnowledgeService(knowledgeLogicImpl, knowledgeBaseLogicImpl)

	// 6. gRPC server 注册回调（同时供聚合 Server 与 app 包测试入口复用）
	registerGRPC := func(g *grpc.Server) {
		knowledgev1.RegisterKnowledgeServiceServer(g, grpcserver.NewKnowledgeGrpcServer(retrievalScopeLogicImpl, knowledgeBaseLogicImpl))
	}

	// 7. 装配聚合 HTTP + gRPC Server
	httpServer := infra.NewHTTPServer(c.Http, func(s *rest.Server) { api.RegisterHandlers(s, knowledgeService) })
	rpcServer := infra.NewRPCServer(c.Rpc, registerGRPC)

	return infra.NewServer(httpServer, rpcServer), nil
}
