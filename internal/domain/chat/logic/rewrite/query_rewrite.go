package rewrite

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// 触发 LLM 改写的问题长度阈值
const (
	shortQuestionLen   = 5  // 无历史上下文时的长度阈值
	historyQuestionLen = 10 // 存在历史上下文时的长度阈值
)

// QueryRewriteImpl 问题改写逻辑实现
type QueryRewriteImpl struct {
	chatModel      model.ChatModel
	promptTemplate adapter.PromptRenderer
	options        []model.Option
}

// NewQueryRewriteImpl 创建问题改写逻辑实例
func NewQueryRewriteImpl(svcCtx *svc.ServiceContext, chatModel model.ChatModel,
	promptTemplate adapter.PromptRenderer) *QueryRewriteImpl {
	return &QueryRewriteImpl{
		chatModel:      chatModel,
		promptTemplate: promptTemplate,
		options: []model.Option{
			model.WithTemperature(svcCtx.Config.Chat.Rewrite.Temperature),
			model.WithTopP(svcCtx.Config.Chat.Rewrite.TopP),
		},
	}
}

// Rewrite 改写问题（结合历史上下文）
// 流程：空问题直接返回 -> 判断是否需要改写 -> 不需要则返回原问题 -> 需要则 LLM 改写，任一步失败都回退到原问题
func (q *QueryRewriteImpl) Rewrite(ctx context.Context, question, historySummary string) (*vo.QuestionRewriteResult, error) {
	question = strutil.Trim(question)

	// 空问题直接返回空结果
	if question == "" {
		return &vo.QuestionRewriteResult{}, nil
	}

	// 预计算兜底结果，用于快速返回
	fallback := &vo.QuestionRewriteResult{RewrittenQuestion: question}

	if !needsRewrite(question, historySummary) {
		logx.Infof("RAG 改写跳过: question='%s'", question)
		return fallback, nil
	}

	// 渲染提示词
	templateVars := map[string]any{
		"history":  utils.BlankToDefault(historySummary, "无历史上下文"),
		"question": question,
	}
	promptText, err := q.promptTemplate.Render(enum.ChatQueryRewrite, templateVars)
	if err != nil {
		logx.Warnf("RAG 改写失败，回退到原问题: question='%s', err=%v", question, err)
		return fallback, nil
	}

	// 调用 LLM 生成改写结果
	raw, err := q.chatModel.GenerateWithTrace(ctx, enum.ChatStageRewrite, "", promptText, q.options...)
	if err != nil {
		logx.Warnf("RAG 改写失败，回退到原问题: question='%s', err=%v", question, err)
		return fallback, nil
	}

	// 解析 LLM 输出
	var payload struct {
		Rewrite string `json:"rewrite"`
	}
	if err = utils.Unmarshal(raw, &payload); err != nil {
		logx.Errorf("RAG 改写结果不可用，回退到原问题: question='%s', raw='%s'", question, utils.Trim(raw))
		return fallback, nil
	}

	// 规范化改写结果
	rewrite := utils.Trim(payload.Rewrite)
	if rewrite == "" {
		logx.Errorf("RAG 改写结果不可用，回退到原问题: question='%s', raw='%s'", question, utils.Trim(raw))
		return fallback, nil
	}

	logx.Infof("RAG 改写完成: question='%s', rewritten='%s'", question, rewrite)

	return &vo.QuestionRewriteResult{RewrittenQuestion: rewrite, RawModelOutput: raw}, nil
}

// needsRewrite 判断是否需要 LLM 改写，无历史上下文时仅过短问题需要改写；存在历史上下文时放宽阈值，以完成指代消解与省略补全
func needsRewrite(question, historySummary string) bool {
	maxLen := shortQuestionLen
	if utils.IsNotBlank(historySummary) {
		maxLen = historyQuestionLen
	}
	return utils.Len(question) < maxLen
}
