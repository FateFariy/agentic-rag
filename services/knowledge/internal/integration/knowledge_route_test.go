package integration

import (
	"context"
	"testing"
	"time"

	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route/rank"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"

	knowroute "github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route"
)

// routeDebugInput 知识路由调试入参，运行前手动填写
type routeDebugInput struct {
	ConversationId             string   // 会话ID
	ExchangeId                 int64    // 对话轮次ID
	Question                   string   // 原始问题
	RewriteQuestion            string   // 改写问题，可留空
	KnowledgeBaseSelectionMode string   // 知识库选择模式：none/all/selected
	SelectedKnowledgeBaseIds   []int64  // 选中的知识库ID，仅在 selected 模式下生效
	SelectedKnowledgeBaseNames []string // 选中的知识库名称，仅用于路由跟踪记录
	AllowedDocumentIds         []int64  // 允许路由的文档ID，留空则按知识库范围
	SelectedDocumentId         int64    // 用户显式选中的文档ID，用于统计 top3 命中
}

// knowledgeRouteInput 调试用例入参，修改此处后直接运行测试即可观察执行过程
var knowledgeRouteInput = routeDebugInput{
	ConversationId:             "debug-route-0001",
	ExchangeId:                 1,
	Question:                   "购买微型无人机必须实名登记吗？",
	RewriteQuestion:            "",
	KnowledgeBaseSelectionMode: enum.KbSelectionModeSelected,
	SelectedKnowledgeBaseIds:   []int64{1},
	SelectedKnowledgeBaseNames: []string{"测试"},
	AllowedDocumentIds:         []int64{2094078435868545024, 2094080550238162944, 2094084939745071104, 2094085251184726016},
	SelectedDocumentId:         0,
}

// TestKnowledgeRouteImpl_Route 知识路由集成测试
// 按 bootstrap 的装配方式注入真实依赖（MySQL / Minio / Milvus / Embedding），输出各阶段结果便于调试
func TestKnowledgeRouteImpl_Route(t *testing.T) {
	router := newKnowledgeRouter()
	routeCtx := newDebugRouteContext(knowledgeRouteInput)

	t.Logf("入参: %s", utils.ToCompactJSON([]routeDebugInput{knowledgeRouteInput}))

	start := time.Now()
	decision, err := router.Route(context.Background(), routeCtx)
	if err != nil {
		t.Fatalf("知识路由执行失败: %v", err)
	}
	t.Logf("路由耗时: %v", time.Since(start))

	logRouteContext(t, routeCtx)
	logRouteDecision(t, decision)

	if decision.RouteStatus == enum.RouteStatusFailed {
		t.Log("路由未产出候选文档，请检查入参与知识库数据（可用 TestListKnowledgeBases 查看可用ID）")
	}
}

// TestKnowledgeRouteImpl_RecordShadowRoute 影子路由集成测试，验证后台记录链路
func TestKnowledgeRouteImpl_RecordShadowRoute(t *testing.T) {
	router := newKnowledgeRouter()
	routeCtx := newDebugRouteContext(knowledgeRouteInput)

	if err := router.RecordShadowRoute(context.Background(), routeCtx); err != nil {
		t.Fatalf("影子路由记录失败: %v", err)
	}
	t.Logf("影子路由记录完成: conversationId=%s, exchangeId=%d", routeCtx.ConversationId, routeCtx.ExchangeId)
}

// TestListKnowledgeBases 列出可用知识库与可检索文档，辅助填写 routeDebugInput
func TestListKnowledgeBases(t *testing.T) {
	knowledgeRepo := deps.KnowledgeRepo
	documentForKnowledge := deps.DocumentForKnowledge

	bases, err := knowledgeRepo.SelectKnowledgeBases(context.Background())
	if err != nil {
		t.Fatalf("查询知识库列表失败: %v", err)
	}
	for _, base := range bases {
		t.Logf("知识库: id=%d, name=%s, documentCount=%d, retrievableDocumentCount=%d",
			base.ID, base.BaseName, base.DocumentCount, base.RetrievableDocumentCount)

		docs, err := documentForKnowledge.FindRetrievableByKbIds(context.Background(), []int64{base.ID})
		if err != nil {
			t.Errorf("查询知识库[%d]可检索文档失败: %v", base.ID, err)
			continue
		}
		for _, doc := range docs {
			t.Logf("  文档: id=%d, name=%s, lastIndexTaskId=%d", doc.DocumentId, doc.DocumentName, doc.LastIndexTaskId)
		}
	}
}

// newKnowledgeRouter 基于全局 deps 构建知识路由器，并为各 ranker 加上执行过程日志（trace 装饰）。
func newKnowledgeRouter() *knowroute.KnowledgeRouteImpl {
	rankerNames := []string{"DocumentRanker", "ScopeRanker", "TopicRanker"}
	traced := make([]knowroute.Ranker, 0, len(deps.Rankers))
	for i, raw := range deps.Rankers {
		traced = append(traced, &traceRanker{name: rankerNames[i], target: raw})
	}
	return knowroute.NewKnowledgeRouteImpl(deps.KnowledgeRepo, deps.DocumentForKnowledge, traced, knowroute.WithEmbedding(deps.Embedder))
}

// newDebugRouteContext 将调试入参转换为路由上下文
func newDebugRouteContext(in routeDebugInput) *knowroute.Context {
	routeCtx := knowroute.NewRouteContext(in.Question, in.RewriteQuestion, in.SelectedKnowledgeBaseIds, in.AllowedDocumentIds)
	routeCtx.ConversationId = in.ConversationId
	routeCtx.ExchangeId = in.ExchangeId
	routeCtx.KnowledgeBaseSelectionMode = in.KnowledgeBaseSelectionMode
	routeCtx.SelectedDocumentId = in.SelectedDocumentId
	routeCtx.SelectedKnowledgeBaseNames = in.SelectedKnowledgeBaseNames
	return routeCtx
}

// logRouteContext 输出路由上下文的关键信息
func logRouteContext(t *testing.T, routeCtx *knowroute.Context) {
	t.Helper()
	t.Logf("路由文本: %s", routeCtx.RoutingText)
	t.Logf("查询向量: 维度=%d", len(routeCtx.QueryEmbedding))
	t.Logf("生效范围: kbIds=%v, allowedDocumentIds=%v, selectedDocumentId=%d",
		routeCtx.SelectedKnowledgeBaseIds, routeCtx.AllowedDocumentIds, routeCtx.SelectedDocumentId)
}

// logRouteDecision 输出路由决策的完整结果
func logRouteDecision(t *testing.T, decision *vo.KnowledgeRouteDecision) {
	t.Helper()
	t.Logf("决策结果: status=%s, confidence=%.4f, source=%s, reason=%s",
		decision.RouteStatus, decision.Confidence, decision.Source, decision.Reason)
	t.Logf("降级标记: degraded=%v, reasons=%v", decision.IsDegraded, decision.DegradedReasons)
	logRouteCandidates(t, "scope", decision.Scopes)
	logRouteCandidates(t, "topic", decision.Topics)
	logRouteCandidates(t, "document", decision.Documents)
}

// logRouteCandidates 输出候选明细
func logRouteCandidates[T any](t *testing.T, title string, candidates []T) {
	t.Helper()
	t.Logf("%s候选: count=%d, detail=%s", title, len(candidates), utils.ToCompactJSON(candidates))
}

// traceRanker 排名器装饰器，输出每个排名器的耗时与候选变化
type traceRanker struct {
	name   string
	target knowroute.Ranker
}

func (r *traceRanker) Rank(ctx context.Context, rankCtx *rank.Context) error {
	start := time.Now()
	err := r.target.Rank(ctx, rankCtx)
	logx.Infof("知识路由排名[%s]: 耗时=%v, scope=%d, topic=%d, document=%d, diagnostics=%v, err=%v",
		r.name, time.Since(start), len(rankCtx.ScopeCandidates), len(rankCtx.TopicCandidates),
		len(rankCtx.DocumentCandidates), utils.MapKeys(rankCtx.Diagnostics), err)
	return err
}
