package enum

// ============================================================
// ConversationTraceStage 追踪阶段定义
// ============================================================

type ConversationTraceStage struct {
	Code  string
	Name  string
	Order int
}

const (
	memory         = "MEMORY"
	rewrite        = "REWRITE"
	semanticCache  = "SEMANTIC_CACHE"
	agent          = "AGENT"
	route          = "ROUTE"
	ragRetrieve    = "RAG_RETRIEVE"
	answerEvaluate = "ANSWER_EVALUATE"
	recommendation = "RECOMMENDATION"
	cacheWrite     = "CACHE_WRITE"
	finalize       = "FINALIZE"
)

var (
	ConversationTraceStageMemory         = &ConversationTraceStage{memory, "会话记忆", 10}
	ConversationTraceStageRewrite        = &ConversationTraceStage{rewrite, "问题改写", 20}
	ConversationTraceStageSemanticCache  = &ConversationTraceStage{semanticCache, "语义缓存查询", 30}
	ConversationTraceStageRoute          = &ConversationTraceStage{route, "路由判定", 40}
	ConversationTraceStageAgent          = &ConversationTraceStage{agent, "Agent执行", 50}
	ConversationTraceStageRAGRetrieve    = &ConversationTraceStage{ragRetrieve, "RAG 检索", 50}
	ConversationTraceStageAnswerEvaluate = &ConversationTraceStage{answerEvaluate, "回答评估", 60}
	ConversationTraceStageRecommendation = &ConversationTraceStage{recommendation, "推荐问题", 70}
	ConversationTraceStageCacheWrite     = &ConversationTraceStage{cacheWrite, "缓存写入", 80}
	ConversationTraceStageFinalize       = &ConversationTraceStage{finalize, "收尾归档", 90}
)
