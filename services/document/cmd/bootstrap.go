package main

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/rest"
	"google.golang.org/grpc"

	"github.com/swiftbit/know-agent/pkg/ai"
	"github.com/swiftbit/know-agent/pkg/embedding"
	"github.com/swiftbit/know-agent/pkg/grpc/gen/document/v1"
	knowledgev1 "github.com/swiftbit/know-agent/pkg/grpc/gen/knowledge/v1"
	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/pkg/tokenize"
	"github.com/swiftbit/know-agent/services/document/api"
	"github.com/swiftbit/know-agent/services/document/internal/config"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/analysis"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/chunk"
	chunkllm "github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/chunk/llm"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/chunk/recursive"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/chunk/semantic"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/index"
	graphbuild "github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/index/graph"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/index/graph/community"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/index/graph/extract"
	"github.com/swiftbit/know-agent/services/document/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/services/document/internal/infrastructure/port/rpc"
	"github.com/swiftbit/know-agent/services/document/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/services/document/internal/trigger/consumer"
	grpcserver "github.com/swiftbit/know-agent/services/document/internal/trigger/grpc"
	"github.com/swiftbit/know-agent/services/document/internal/trigger/handler"
)

// Build 装配 document 服务出站访问 knowledge 服务的 gRPC 连接在此建立
func Build(c *config.Config) (*infra.Server, error) {
	embedder := embedding.NewOllamaEmbedder(c.Embedding)
	minioStorage := storage.NewMinioStorage(c.Minio)
	gseTokenizer := tokenize.NewGseTokenizer(c.Gse)
	arkChatModel := ai.NewArkChatModel(context.Background(), c.ChatModel["Ark"])
	chatModel := ai.NewChatModelImpl(arkChatModel, c.ChatModel)
	ollamaModel := ai.NewOllamaModel(c.JudgeModel)
	renderer := ai.NewRendererImpl()
	rocketMQProducer := infra.NewRocketMQMessageProducer(c.MQ)
	db := infra.NewDB(c.Mysql)
	rdb := infra.NewRedisClient(c.Redis)
	neo4jStore := infra.NewNeo4jGraphStore(c.Neo4j)
	milvusClient := infra.NewMilvusClient(context.Background(), c.Milvus)
	knowledgeConn, err := infra.DialGrpc(c.Grpc.KnowledgeAddr)
	if err != nil {
		return nil, fmt.Errorf("dial knowledge service: %w", err)
	}
	knowledgeClient := rpc.NewKnowledgeClient(knowledgev1.NewKnowledgeServiceClient(knowledgeConn))

	documentRepo := persistence.NewDocumentRepository(db, rdb, minioStorage)
	strategyRegistry := newChunkStrategyRegistry(c.Chunk, chatModel, renderer)
	generateImpl := process.NewProfileGenerateImpl(documentRepo)
	analysisChain := analysis.NewAnalysisChain(c.Chunk, chatModel, documentRepo, minioStorage, generateImpl, knowledgeClient)
	reporter := community.NewLLMCommunityReporter(ollamaModel, renderer)
	llmExtractor := extract.NewLLMExtractor(ollamaModel, renderer, gseTokenizer, c.Rag.Graph.ChunkMaxChars)
	milvusIndexer := storage.NewMilvusIndexer(milvusClient, c.Milvus, c.Embedding.Dimensions, c.Embedding.Model, embedder)
	graphStore := storage.NewDocumentGraphStore(neo4jStore)
	graphBuilder := graphbuild.NewBuilder(graphStore, llmExtractor, c.Rag.Graph, reporter, milvusIndexer, knowledgeClient)
	indexChain := index.NewBuildIndexChain(documentRepo, minioStorage, milvusIndexer, milvusIndexer, strategyRegistry, knowledgeClient, gseTokenizer, graphBuilder)
	asyncProcessImpl := process.NewAsyncProcessImpl(documentRepo, analysisChain, indexChain)
	lifecycleLogicImpl := logic.NewLifecycleLogicImpl(rocketMQProducer, minioStorage, documentRepo, knowledgeClient, milvusIndexer, graphStore, c.MQ.ParseTopic, c.MQ.IndexTopic)
	profileLogicImpl := logic.NewProfileLogicImpl(documentRepo, minioStorage, generateImpl)

	parseConsumer := consumer.NewParseDocumentConsumer(c.MQ.Addr, c.MQ.ParseTopic, asyncProcessImpl)
	buildIndexConsumer := consumer.NewBuildIndexConsumer(c.MQ.Addr, c.MQ.IndexTopic, asyncProcessImpl)
	documentService := handler.NewDocumentService(lifecycleLogicImpl, profileLogicImpl)

	queryLogic := logic.NewDocumentQueryLogicImpl(documentRepo)
	registerGRPC := func(g *grpc.Server) {
		documentv1.RegisterDocumentServiceServer(g, grpcserver.NewDocumentGrpcServer(queryLogic))
	}

	httpServer := infra.NewHTTPServer(c.Http, func(s *rest.Server) { api.RegisterHandlers(s, documentService) })
	rpcServer := infra.NewRPCServer(c.Rpc, registerGRPC)

	return infra.NewServer(httpServer, rpcServer, rocketMQProducer, parseConsumer, buildIndexConsumer), nil
}

// newChunkStrategyRegistry 构建分块策略注册表（递归/语义/LLM）
func newChunkStrategyRegistry(chunkConf config.ChunkConf, chatModel ai.ChatModel, renderer ai.PromptRenderer) *chunk.Registry {
	recursiveChunker := recursive.NewChunker(
		recursive.WithMaxChars(chunkConf.RecursiveMaxChars),
		recursive.WithOverlapChars(chunkConf.RecursiveOverlapChars),
	)
	semanticChunker := semantic.NewChunker(
		semantic.WithMaxChars(chunkConf.SemanticMaxChars),
		semantic.WithMinChars(chunkConf.SemanticMinChars),
		semantic.WithSimilarityThreshold(chunkConf.SemanticSimilarityThreshold),
	)
	llmChunker := chunkllm.NewChunker(chatModel, renderer,
		chunkllm.WithLlmMaxChars(chunkConf.LlmMaxChars),
		chunkllm.WithEnabled(chunkConf.LlmEnabled),
	)
	return chunk.NewChunkStrategyRegistry([]chunk.Chunker{recursiveChunker, semanticChunker, llmChunker})
}
