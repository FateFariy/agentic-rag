package main

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/pkg/ai"
	"github.com/swiftbit/know-agent/pkg/embedding"
	documentv1 "github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/pkg/tokenize"
	"github.com/swiftbit/know-agent/services/chat/api"
	"github.com/swiftbit/know-agent/services/chat/internal/config"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/conversation"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/conversation/middleware"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/conversation/tools"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/evaluate"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/memory"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/recommend"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/retrieval"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/retrieval/channel"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/retrieval/fuse"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/rewrite"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/route"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/observability"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/port/agent"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/port/cache"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/port/lock"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/port/rerank"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/port/rpc"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/services/chat/internal/trigger/consumer"
	"github.com/swiftbit/know-agent/services/chat/internal/trigger/handler"
)

// Build 装配 chat 微服务
func Build(c *config.Config) (*infra.Server, error) {
	// 1. 出站 gRPC 客户端（document / knowledge）
	documentConn, err := infra.DialGrpc(c.Grpc.DocumentAddr)
	if err != nil {
		return nil, fmt.Errorf("dial document service: %w", err)
	}
	documentClient := rpc.NewChatDocumentClient(documentv1.NewDocumentServiceClient(documentConn))
	knowledgeConn, err := infra.DialGrpc(c.Grpc.KnowledgeAddr)
	if err != nil {
		return nil, fmt.Errorf("dial knowledge service: %w", err)
	}
	knowledgeClient := rpc.NewKnowledgeClient(knowledgev1.NewKnowledgeServiceClient(knowledgeConn))

	// 2. 基础设施
	db := infra.NewDB(c.Mysql)
	redSync := infra.NewRedSync(infra.NewRedisClient(c.Redis))
	arkChatModel := ai.NewArkChatModel(context.Background(), c.ChatModel["Ark"])
	chatModel := ai.NewChatModelImpl(arkChatModel, c.ChatModel)
	milvus := infra.NewMilvusClient(context.Background(), c.Milvus)
	neo4j := infra.NewNeo4jGraphStore(c.Neo4j)
	embedder := embedding.NewOllamaEmbedder(c.Embedding)
	milvusStorage := storage.NewMilvusStorage(c.Milvus, embedder)
	rocketMQProducer := infra.NewRocketMQMessageProducer(c.MQ)
	redisLock := lock.NewRedisMutexLock(redSync)
	ollamaModel := ai.NewOllamaModel(c.JudgeModel)
	renderer := ai.NewRendererImpl()
	dashScope := rerank.NewDashScope(c.Chat.Rag.Rerank)
	gseTokenizer := tokenize.NewGseTokenizer(c.Gse)
	semanticCacheStore := cache.NewSemanticCache(milvus, db, embedder, c.Chat.SemanticCache.Collection)
	graphRetriever := storage.NewGraphRetriever(neo4j, gseTokenizer)
	chatRepo := persistence.NewChatRepository(db)

	// 3. 领域逻辑
	observability.RegisterConversationTraceRecorder(chatRepo)
	observability.RegisterRagEvalMetricsHandler(chatRepo)
	observability.RegisterModelUsageHandler(c.ChatModel)

	recommender := recommend.NewQuestionRecommendImpl(c.Chat.Recommendation, renderer, ollamaModel)
	channels := []retrieval.Retrieval{
		channel.NewVectorRetrievalChannel(milvusStorage),
		channel.NewKeywordRetrievalChannel(milvusStorage),
		channel.NewGraphRetrievalChannel(graphRetriever),
	}
	rrfFusion := fuse.NewRRFFusion()
	retrievalEngine := retrieval.NewRetrievalEngine(c.Chat.Rag.ParentEvidenceMaxChars, chatRepo, dashScope, channels, documentClient, rrfFusion)
	compressionStrategy := memory.NewSummaryCompressionStrategy(c.Chat.Memory, chatRepo, chatModel, renderer)
	rewriteImpl := rewrite.NewQueryRewriteImpl(c.Chat.Rewrite, ollamaModel, renderer)
	chainRuntime := conversation.NewRuntimeRegistry()
	documentRouter := route.NewDocumentRouter(documentClient, nil)
	contextInjectMiddleware := middleware.NewContextInjectMiddleware(c.Chat.Rag)
	structureTool := tools.NewRouteDocumentStructureTool(documentRouter)
	knowledgeBaseSearchTool := tools.NewKnowledgeBaseSearchTool(retrievalEngine, c.Chat.Rag)
	agentRunner := agent.NewEinoAgentRunner(arkChatModel,
		agent.WithMiddleware(contextInjectMiddleware),
		agent.WithTools(structureTool),
		agent.WithTools(knowledgeBaseSearchTool),
	)
	stages := []conversation.Stage{
		conversation.NewStartStage(chatRepo, chainRuntime, redisLock),
		conversation.NewMemoryLoadStage(compressionStrategy, chatRepo, c.Chat.Rag),
		conversation.NewQueryRewriteStage(rewriteImpl, c.Chat.Rewrite, c.Chat.Rag),
		conversation.NewRouteStage(chatRepo, knowledgeClient, documentClient),
		conversation.NewSemanticCacheStage(semanticCacheStore, ollamaModel, renderer, c.Chat.SemanticCache),
		conversation.NewAgentStage(agentRunner),
		conversation.NewCacheWriteStage(semanticCacheStore, rocketMQProducer, c.Chat.SemanticCache),
		conversation.NewAnswerEvaluateStage(NewAnswerEvaluators(ollamaModel, renderer, embedder)),
		conversation.NewRecommendStage(chatRepo, compressionStrategy, recommender, c.Chat.Recommendation),
		conversation.NewEndStage(chatRepo),
	}
	chain := conversation.NewChain(chatRepo, redisLock, compressionStrategy, chainRuntime, stages)
	conversationLogicImpl := logic.NewConversationLogicImpl(chatRepo, knowledgeClient, compressionStrategy, redisLock, chain)

	// 4. 应用层 Service 与聚合 Server
	semanticCacheWriteConsumer := consumer.NewSemanticCacheWriteConsumer(semanticCacheStore, c.MQ.Addr, c.Chat.SemanticCache.WriteTopic)
	chatService := handler.NewChatService(conversationLogicImpl)
	httpServer := infra.NewHTTPServer(c.Http, func(s *rest.Server) { api.RegisterHandlers(s, chatService) })

	return infra.NewServer(httpServer, nil, rocketMQProducer, semanticCacheWriteConsumer), nil
}

// NewAnswerEvaluators 构建回答质量评估器集合（chat 服务使用）。
func NewAnswerEvaluators(llm ai.ChatModel, renderer ai.PromptRenderer, emb *embedding.OllamaEmbedder) []evaluate.Evaluator {
	return []evaluate.Evaluator{
		evaluate.NewAnswerFaithfulnessEvaluator(llm, renderer),
		evaluate.NewAnswerRelevancyEvaluator(llm, renderer, emb),
		evaluate.NewContextPrecisionEvaluator(llm, renderer),
		evaluate.NewContextRecallEvaluator(llm, renderer),
	}
}
