package community

import (
	"context"
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/pkg/ai"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
)

const (
	maxReportMembers     = 40  // 社区报告成员上限（token 预算）
	communityReportChars = 600 // 社区报告摘要最大字符数
)

// LLMCommunityReporter M3：基于 LLM 的社区报告生成器。
// 社区报告为效果增强：生成失败时由 builder 回退抽取式摘要，不阻断图谱构建。
type LLMCommunityReporter struct {
	chatModel ai.ChatModel
	renderer  ai.PromptRenderer
}

// NewLLMCommunityReporter 创建 LLM 社区报告器
func NewLLMCommunityReporter(chatModel ai.ChatModel, renderer ai.PromptRenderer) *LLMCommunityReporter {
	return &LLMCommunityReporter{chatModel: chatModel, renderer: renderer}
}

// Report 渲染模板并调用 LLM 生成单个社区摘要
func (r *LLMCommunityReporter) Report(ctx context.Context, community *entity.GraphCommunity, members []*entity.GraphNode) (string, error) {
	if community == nil || len(members) == 0 {
		return "", fmt.Errorf("社区或成员为空，跳过 LLM 报告")
	}
	// 成员按重要性降序截断，控制 prompt token 预算
	top := utils.Limit(members, maxReportMembers)
	lines := utils.Map(top, func(n *entity.GraphNode) string {
		if n == nil {
			return ""
		}
		return fmt.Sprintf("%s(%s)", n.Name, n.EntityType)
	})
	membersText := strings.Join(utils.FilterBlank(lines), "\n")
	topMembers := strings.Join(utils.Map(utils.Limit(top, 3), func(n *entity.GraphNode) string { return n.Name }), "、")
	prompt, err := r.renderer.Render(enum.GraphCommunityReport, map[string]any{
		"members":    membersText,
		"topMembers": topMembers,
	})
	if err != nil {
		return "", fmt.Errorf("render community report prompt: %w", err)
	}
	resp, err := r.chatModel.Generate(ctx, "", prompt)
	if err != nil {
		return "", fmt.Errorf("llm generate community report: %w", err)
	}
	resp = strings.TrimSpace(resp)
	if resp == "" {
		return "", fmt.Errorf("社区报告为空")
	}
	return utils.ClipHead(resp, communityReportChars), nil
}
