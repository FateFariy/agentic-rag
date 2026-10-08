package conversation

import (
	"context"
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/pkg/ai"
	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/chat/internal/config"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/vo"
)

// SemanticCacheStage 语义缓存命中判定阶段。
//
// 位置：QueryRewrite 之后、Route 之前。
// 理由：问题改写必须执行（指代消解/歧义消除），缓存判定基于「改写后的问题」语义向量，
// 命中即语义真正等价。领域层不负责向量化，由 SemanticCacheStore 实现层完成。
//
// 命中判定采用分区间策略（设计方案：无 Rerank + 大模型终判）：
//  1. 向量候选召回：TopK（≥召回阈值），存储层不判定命中；
//  2. 最高候选得分 ≥ 直接命中阈值（高置信区）→ 直接命中 Top1，零大模型开销；
//  3. 最高候选得分处于灰色区间 → JudgeEnabled 时大模型批量终判：
//     · 命中（含置信度兜底地板校验）→ 复用答案与检索结果（或按配置仅复用检索）；
//     · 未命中（含置信度不足）→ 软复用检索结果：挂载最高相似度候选的检索结果作为参考，
//     但不标记缓存命中，是否采用由 Agent 自行决定（可基于该结果作答，亦可重新检索）；
//  4. 低于召回阈值 → 直接未命中，走完整下游链路。
//
// 命中后：把缓存的必要执行产物回填进当前执行计划并打标；后续 Route / AnswerEvaluate 早退。
// 软复用（未命中但挂载检索结果）不打标，Agent 全程运行、自行决定是否采用挂载的检索结果。
type SemanticCacheStage struct {
	store    SemanticCacheStore
	llm      ai.ChatModel
	renderer ai.PromptRenderer
	*cacheOptions
}

type cacheOptions struct {
	enabled             bool               // 是否启用语义缓存
	similarityThreshold float32            // 向量召回粗筛阈值：低于该分直接未命中
	directHitThreshold  float32            // 直接命中阈值：最高候选达到该分跳过 LLM 判定
	recallTopK          int                // 向量召回候选数
	confidenceFloor     float32            // LLM 判定置信度兜底地板
	judgeEnabled        bool               // 灰色区间是否启用大模型等价终判
	judgeTemperature    float32            // 大模型判定温度
	reuseStrategy       enum.ReuseStrategy // 复用策略
}

var _ Stage = (*SemanticCacheStage)(nil)

var _ ConditionalStage = (*SemanticCacheStage)(nil)

func NewSemanticCacheStage(store SemanticCacheStore, judgeLLM ai.ChatModel, renderer ai.PromptRenderer, sc config.SemanticCacheConf) *SemanticCacheStage {
	return &SemanticCacheStage{
		store:    store,
		llm:      judgeLLM,
		renderer: renderer,
		cacheOptions: &cacheOptions{
			enabled:             sc.Enabled,
			similarityThreshold: sc.SimilarityThreshold,
			directHitThreshold:  sc.DirectHitThreshold,
			recallTopK:          sc.RecallTopK,
			confidenceFloor:     sc.ConfidenceFloor,
			judgeEnabled:        sc.JudgeEnabled,
			judgeTemperature:    sc.JudgeTemperature,
			reuseStrategy:       utils.Ternary(sc.ReuseAnswer, enum.ReuseAnswerAndRetrieval, enum.ReuseRetrievalOnly),
		},
	}
}

func (s *SemanticCacheStage) Name() string {
	return enum.ConversationTraceStageSemanticCache.Name
}

func (s *SemanticCacheStage) Order() int {
	return enum.ConversationTraceStageSemanticCache.Order
}

// ShouldExecute 仅当语义缓存已启用、执行计划就绪且非实时/时间敏感时执行
func (s *SemanticCacheStage) ShouldExecute(ctx context.Context, convCtx *Context) bool {
	if !s.enabled || s.store == nil {
		return false
	}
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return false
	}
	return !execPlan.RequiresRealTimeSearch && !execPlan.RequiresCurrentDateAnchoring
}

func (s *SemanticCacheStage) Execute(ctx context.Context, convCtx *Context) error {
	convCtx.cache = &semanticCacheCtx{strategy: s.reuseStrategy}
	execPlan := convCtx.ExecutionPlan.Load()
	queryText := utils.Trim(execPlan.RewriteQuestion)

	// 启动语义缓存追踪阶段
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageSemanticCache, &vo.StageInput{
		SummaryText: "正在查询语义缓存。",
		Snapshot: map[string]any{
			"rewriteQuestion":    queryText,
			"recallThreshold":    s.similarityThreshold,
			"directHitThreshold": s.directHitThreshold,
			"recallTopK":         s.recallTopK,
			"judgeEnabled":       s.judgeEnabled,
			"reuseStrategy":      s.reuseStrategy,
		},
	})

	// 故障降级：查询失败视为未命中，不阻塞主链路
	candidates, err := s.store.Search(ctx, &SearchInput{
		Scope:     buildCacheScope(convCtx),
		QueryText: execPlan.RewriteQuestion,
		Threshold: s.similarityThreshold,
		TopK:      s.recallTopK,
	})
	if err != nil {
		logx.Warnf("语义缓存查询失败，降级未命中: conversationId=%s, error=%v", convCtx.ConversationId, err)
		ctx = vo.OnError(ctx, "语义缓存查询失败，降级未命中。", err)
		convCtx.cache.MarkCacheMiss()
		return nil
	}

	hit, isHit := s.decideHit(ctx, execPlan.RewriteQuestion, candidates)
	if hit == nil {
		convCtx.cache.MarkCacheMiss()
		ctx = vo.OnEnd(ctx, &vo.StageOutput{
			SummaryText: "语义缓存未命中。",
			Snapshot: map[string]any{
				"hit":             false,
				"candidateCount":  len(candidates),
				"rewriteQuestion": queryText,
			},
		})
		logx.Infof("语义缓存未命中: conversationId=%s, rewriteQuestion='%s', candidateCount=%d",
			convCtx.ConversationId, queryText, len(candidates))
		return nil
	}

	// 轻量校验：检索结果合法性 + Prompt 完整性（防配置漂移导致复用了不符预期的产物）
	if !hit.Entry.Validate() {
		logx.Warnf("语义缓存条目校验失败，降级未命中: conversationId=%s", convCtx.ConversationId)
		convCtx.cache.MarkCacheMiss()
		ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "语义缓存条目校验失败，降级未命中。"})
		return nil
	}

	// 挂载最高相似度候选的检索结果：是否复用由 Agent 自行决定。
	execPlan.RetrievalResult = hit.Entry.Execution.RetrievalResult
	if !isHit {
		// 软复用：大模型终判未命中，但提供检索结果作为参考，不标记缓存命中，后续 Agent 全程运行并自行判断是否采用该检索结果（亦可重新检索）
		convCtx.cache.MarkCacheMiss()
		ctx = vo.OnEnd(ctx, &vo.StageOutput{
			SummaryText: "语义缓存未直接命中，挂载参考检索结果（由模型决定是否采用）。",
			Snapshot: map[string]any{
				"hit":             false,
				"softReuse":       true,
				"entryId":         hit.Entry.ID,
				"similarity":      hit.Similarity,
				"rewriteQuestion": queryText,
			},
		})
		logx.Infof("语义缓存软复用（未直接命中）: conversationId=%s, entryId=%d, similarity=%.4f",
			convCtx.ConversationId, hit.Entry.ID, hit.Similarity)
		return nil
	}

	// 真正命中：打标并复用，后续 Route / AnswerEvaluate 早退
	convCtx.cache.MarkCacheHit(hit, s.reuseStrategy)
	if s.reuseStrategy == enum.ReuseAnswerAndRetrieval {
		if err = convCtx.PublishThinking("分析用户意图... 该请求与历史成功执行的任务高度重合，置信度达标。已有成熟的回答草稿可以复用，跳过重复计算，直接生成最终回复。"); err != nil {
			logx.Warnf("发布思考过程失败: conversationId=%s, error=%v", convCtx.ConversationId, err)
		}
	}

	ctx = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "语义缓存命中，复用已有执行产物。",
		Snapshot: map[string]any{
			"hit":           true,
			"entryId":       hit.Entry.ID,
			"similarity":    hit.Similarity,
			"confidence":    hit.Confidence,
			"reason":        hit.Reason,
			"reuseStrategy": s.reuseStrategy,
			"answerLength":  len(hit.Entry.AnswerDraft),
		},
	})
	logx.Infof("语义缓存命中: conversationId=%s, entryId=%d, similarity=%.4f, confidence=%.4f, reuseStrategy=%d",
		convCtx.ConversationId, hit.Entry.ID, hit.Similarity, hit.Confidence, s.reuseStrategy)

	return nil
}

// decideHit 分区间命中判定，返回命中的缓存条目以及是否为「真正命中」：
//   - 无候选或最高得分低于召回阈值 → 未命中（nil, false）；
//   - 最高得分 ≥ 直接命中阈值（高置信区）→ 直接命中 Top1（true），置信度记为 1.0；
//   - 灰色区间：JudgeEnabled 时大模型批量终判，命中为 true，未命中降级为软复用（false）。
func (s *SemanticCacheStage) decideHit(ctx context.Context, query string, candidates []*CacheHit) (*CacheHit, bool) {
	if len(candidates) == 0 {
		return nil, false
	}
	best := candidates[0]
	if best.Similarity >= s.directHitThreshold {
		best.Confidence = 1
		return best, true
	}
	if s.judgeEnabled {
		return s.judgeCacheEquivalence(ctx, query, candidates)
	}
	return nil, false
}

// buildCacheScope 仅包含数据隔离维度
func buildCacheScope(convCtx *Context) *vo.CacheScope {
	snap := convCtx.KnowledgeBaseSelectionSnapshot
	return &vo.CacheScope{
		ChatMode:           convCtx.ChatMode,
		AllowedDocumentIds: snap.SelectedDocumentIds(),
		AllowedTaskIds:     snap.SelectedTaskIds(),
		KnowledgeBaseIds:   snap.SelectedKnowledgeBaseIds,
	}
}

// cacheEquivalenceOutput 大模型语义等价终判输出
type cacheEquivalenceOutput struct {
	Hit        bool    `json:"hit"`        // 是否命中
	Index      int     `json:"index"`      // 命中候选在候选列表中的 1-based 序号
	Reason     string  `json:"reason"`     // 命中原因
	Confidence float64 `json:"confidence"` // 置信度
}

// judgeCacheEquivalence 对灰色区间候选做批量等价终判：
//   - 模型判定命中且置信度 ≥ confidenceFloor，且命中条目在候选池内 → 返回该候选（isHit=true）；
//   - 模型判定未命中（或置信度不足）→ 软复用（isHit=false）：返回最高相似度候选的检索结果作为参考，
//     不标记命中，是否采用由 Agent 自行决定；
//   - 调用失败、命中索引越界等异常情况 → 保守未命中（nil, false），避免模型幻觉误召回。
func (s *SemanticCacheStage) judgeCacheEquivalence(ctx context.Context, query string, candidates []*CacheHit) (*CacheHit, bool) {
	if s.llm == nil || s.renderer == nil || len(candidates) == 0 {
		return nil, false
	}
	output, err := s.renderAndJudge(ctx, query, candidates)
	if err != nil {
		logx.Warnf("语义缓存大模型终判失败，降级未命中: query='%s', error=%v", query, err)
		return nil, false
	}
	if !output.Hit || output.Confidence < float64(s.confidenceFloor) {
		// 终判未命中（或置信度不足）：不标记命中，仅挂载最高相似度候选的检索结果供 Agent 参考决策。
		best := candidates[0]
		logx.Infof("语义缓存大模型终判未命中，软复用检索结果（不标记命中）: query='%s', hit=%t, confidence=%.4f, floor=%.4f, similarity=%.4f",
			query, output.Hit, output.Confidence, s.confidenceFloor, best.Similarity)
		return best, false
	}
	targetIndex := output.Index
	if targetIndex < 1 || targetIndex > len(candidates) {
		logx.Warnf("语义缓存大模型终判命中索引越界，按未命中处理: index=%d, candidateCount=%d", targetIndex, len(candidates))
		return nil, false
	}
	cand := candidates[targetIndex-1]
	cand.Confidence = float32(output.Confidence)
	cand.Reason = output.Reason
	return cand, true
}

// renderAndJudge 渲染判定模板并调用大模型，返回结构化判定结果
func (s *SemanticCacheStage) renderAndJudge(ctx context.Context, query string, candidates []*CacheHit) (*cacheEquivalenceOutput, error) {
	lines := make([]string, 0, len(candidates))
	for i, cand := range candidates {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, strings.TrimSpace(cand.Entry.QueryText)))
	}
	prompt, err := s.renderer.Render(enum.SemanticCacheJudge, map[string]any{
		"currentQuery":   query,
		"candidatesList": strings.Join(lines, "\n"),
	})
	if err != nil {
		return nil, fmt.Errorf("渲染语义缓存判定模板失败: %w", err)
	}
	response, err := s.llm.Generate(ctx, "", prompt, ai.WithFunction("judge"), ai.WithTemperature(s.judgeTemperature), ai.WithThink(false))
	if err != nil {
		return nil, fmt.Errorf("调用大模型判定失败: %w", err)
	}
	var output cacheEquivalenceOutput
	if err = utils.Unmarshal(response, &output); err != nil {
		return nil, fmt.Errorf("解析大模型判定结果失败: %w", err)
	}
	logx.Infof("语义缓存大模型终判结果: hit=%t, index=%d, confidence=%.4f, reason='%s'", output.Hit, output.Index, output.Confidence, output.Reason)
	return &output, nil
}
