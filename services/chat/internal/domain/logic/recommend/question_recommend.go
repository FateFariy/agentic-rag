package recommend

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/pkg/ai"
	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/chat/internal/config"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/enum"
)

const (
	maxRecommendations = 3
)

// QuestionRecommendImpl 追问推荐实现
type QuestionRecommendImpl struct {
	promptTemplate      ai.PromptRenderer
	chatModel           ai.ChatModel
	historyPreviewTurns int
}

func NewQuestionRecommendImpl(recommendConf config.RecommendationConf, promptTemplate ai.PromptRenderer, chatModel ai.ChatModel) *QuestionRecommendImpl {
	return &QuestionRecommendImpl{
		chatModel:           chatModel,
		promptTemplate:      promptTemplate,
		historyPreviewTurns: recommendConf.HistoryPreviewTurns,
	}
}

// Generate 生成推荐追问
func (r *QuestionRecommendImpl) Generate(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange) ([]string, error) {
	if utils.IsBlank(answer) {
		return nil, nil
	}

	// 渲染提示词模板
	userPrompt, err := r.promptTemplate.Render(enum.RecommendationUser, map[string]any{
		"recentContext": r.buildRecentContext(recentExchanges),
		"question":      question,
		"answer":        answer,
	})
	if err != nil {
		return nil, err
	}

	// 调用LLM生成推荐
	content, err := r.chatModel.GenerateWithTrace(ctx, enum.ChatStageRecommend, "", userPrompt, ai.WithThink(false))
	if err != nil {
		return nil, err
	}
	if utils.IsBlank(content) {
		return nil, nil
	}

	// 解析JSON数组
	var result []string
	if err = utils.Unmarshal(content, &result); err != nil {
		logx.Warnf("解析推荐问题失败: content=%s, err=%v", content, err)
		return nil, err
	}

	// 去重并限制数量
	result = utils.FilterMapUniqueLimit(result, maxRecommendations, func(item string) (string, string, bool) {
		return item, utils.Trim(item), utils.IsNotBlank(item)
	})

	return result, nil
}

// buildRecentContext 构建最近对话上下文
func (r *QuestionRecommendImpl) buildRecentContext(recentExchanges []*entity.ChatExchange) string {
	if len(recentExchanges) == 0 {
		return ""
	}

	var sb strings.Builder
	historyTurns := max(r.historyPreviewTurns, 3)
	startIndex := max(len(recentExchanges)-historyTurns, 0)
	for i := startIndex; i < len(recentExchanges); i++ {
		exchange := recentExchanges[i]
		sb.WriteString("用户：")
		sb.WriteString(exchange.Question)
		sb.WriteString("\n")
		if utils.IsNotBlank(exchange.Answer) {
			sb.WriteString("助手：")
			sb.WriteString(exchange.Answer)
			sb.WriteString("\n")
		}
	}

	return utils.Trim(sb.String())
}
