package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/swiftbit/know-agent/pkg/config"
	"github.com/swiftbit/know-agent/pkg/emb"
	"github.com/swiftbit/know-agent/pkg/svc"
	documentapp "github.com/swiftbit/know-agent/services/document/app"
	knowroute "github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route/rank"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/storage"
)

// configPath 集成测试使用的配置文件路径，可按需切换到 config-dev.yaml / config-prod.yaml。
const configPath = `E:\GoCode\ragent-convert\know-agent\etc\config-prod.yaml`

// integrationDeps 集中挂载 knowledge 路由集成测试所需的真实依赖，
// 按 knowledge bootstrap 的装配方式构建（MySQL / Embedding / 词法检索桩）。
//
// document 侧依赖：测试需要 document gRPC 返回文档元数据/画像，经 document 服务公开
// 装配入口 services/document/app 在本进程内拉起其查询 gRPC server。
type integrationDeps struct {
	svcCtx               *svc.ServiceContext
	Embedder             *emb.OllamaEmbedder
	KnowledgeRepo        *persistence.KnowledgeRepositoryImpl
	DocumentForKnowledge *gateway.KnowledgeDocumentClient
	Rankers              []knowroute.Ranker

	documentGrpcServer *grpc.Server
	documentGrpcConn   *grpc.ClientConn
}

// deps 包级全局依赖，由 init 在测试启动时初始化一次。
var deps *integrationDeps

func init() {
	var c config.Config
	conf.MustLoad(configPath, &c)
	svcCtx := svc.NewServiceContext(&c)
	deps = buildDeps(svcCtx)
}

// TestMain 在全部测试结束后统一释放连接（gRPC / MySQL / Redis / Milvus）。
func TestMain(m *testing.M) {
	code := m.Run()

	if deps != nil {
		if deps.documentGrpcConn != nil {
			_ = deps.documentGrpcConn.Close()
		}
		if deps.documentGrpcServer != nil {
			deps.documentGrpcServer.Stop()
		}
	}
	if deps != nil && deps.svcCtx != nil {
		if db, err := deps.svcCtx.Db.DB(); err == nil {
			_ = db.Close()
		}
		_ = deps.svcCtx.Rdb.Close()
		_ = deps.svcCtx.Milvus.Close(context.Background())
	}

	os.Exit(code)
}

// buildDeps 按 knowledge bootstrap 的装配顺序构建测试依赖。
func buildDeps(svcCtx *svc.ServiceContext) *integrationDeps {
	embedder := emb.NewOllamaEmbedder(svcCtx.Config.Embedding)
	elasticStorage := storage.NewElasticStorage()

	knowledgeRepo := persistence.NewKnowledgeRepository(svcCtx.Db)

	// 经 document 服务公开装配入口本地拉起真实的 document 查询 gRPC server，
	// knowledge 侧通过自己的出站客户端调用，与生产环境的跨进程调用路径保持一致
	// （端口用 0 由系统分配，避免与本机服务冲突）。
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Errorf("listen document grpc: %w", err))
	}
	documentSrv := grpc.NewServer()
	documentapp.RegisterQueryGRPC(documentSrv, svcCtx.Config)
	go func() { _ = documentSrv.Serve(lis) }()

	documentConn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("dial document grpc: %w", err))
	}
	documentForKnowledge := gateway.NewKnowledgeDocumentClient(documentConn)

	opts := []rank.Option{rank.WithLexicalIndex(elasticStorage), rank.WithEmbedding(embedder)}
	rankers := []knowroute.Ranker{
		rank.NewDocumentRanker(knowledgeRepo, documentForKnowledge, opts...),
		rank.NewScopeRanker(knowledgeRepo, documentForKnowledge, opts...),
		rank.NewTopicRanker(knowledgeRepo, documentForKnowledge, opts...),
	}

	return &integrationDeps{
		svcCtx:               svcCtx,
		Embedder:             embedder,
		KnowledgeRepo:        knowledgeRepo,
		DocumentForKnowledge: documentForKnowledge,
		Rankers:              rankers,
		documentGrpcServer:   documentSrv,
		documentGrpcConn:     documentConn,
	}
}
