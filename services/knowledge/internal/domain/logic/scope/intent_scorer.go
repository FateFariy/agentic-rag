package scope

import (
	"context"
	"strings"
	"time"

	"github.com/swiftbit/know-agent/pkg/ai"
	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/adapter"
)

const (
	defaultIntentScoreTimeout = 8 * time.Second          // 意图打分的默认超时时间
	intentScoreTemplateName   = "knowledge-topic-intent" // 主题意图打分模板名
)

// intentScoreResult 大模型意图打分输出结构：{"scores":[{"topicId":123,"score":0.73}]}
type intentScoreResult struct {
	Scores []intentScoreItem `json:"scores"`
}

// intentScoreItem 单个节点的打分项
type intentScoreItem struct {
	TopicId int64   `json:"topicId"`
	Score   float64 `json:"score"`
}

// KnowledgeIntentScorer 知识主题意图打分器：一次性把候选主题节点批量交给大模型打分，
// 所有外部依赖（节点加载、提示词渲染、模型调用、结果解析）失败均降级为 nil，不返回 error，调用方据此回退全局检索
type KnowledgeIntentScorer struct {
	repo      adapter.KnowledgeRepository
	chatModel ai.ChatModel
	renderer  ai.PromptRenderer
	timeout   time.Duration
}

// NewKnowledgeIntentScorer 创建主题意图打分器
func NewKnowledgeIntentScorer(repo adapter.KnowledgeRepository, chatModel ai.ChatModel, renderer ai.PromptRenderer, timeout time.Duration) *KnowledgeIntentScorer {
	if timeout <= 0 {
		timeout = defaultIntentScoreTimeout
	}
	return &KnowledgeIntentScorer{
		repo:      repo,
		chatModel: chatModel,
		renderer:  renderer,
		timeout:   timeout,
	}
}

// Score 对指定知识库范围内的候选主题节点做意图相关性打分。
//
// 降级语义（均返回 nil，表示结果不可信，消费方应回退 GLOBAL）：
//  1. question 空白或 kbIds 为空；
//  2. 节点加载失败或候选节点为空；
//  3. 提示词渲染失败；
//  4. 模型调用失败/超时；
//  5. 模型输出无法解析为打分 JSON 对象。
//
// 正常返回时结果长度与节点列表严格相等、顺序与节点列表一致：
// LLM 未返回的节点补 0；重复 topicId 取最大分；分数 clamp 到 [0,1]。
// 即使全部节点分数为 0（可识别但无命中），也返回完整长度的切片，
// 与第 4/5 步的不可信空结果 nil 相区别。
func (r *KnowledgeIntentScorer) Score(ctx context.Context, question string, kbIds []int64) []*NodeScore {
	// 1. 入参防御：空白问题或空知识库范围直接返回
	if strings.TrimSpace(question) == "" || len(kbIds) == 0 {
		return nil
	}

	// 2. 加载候选主题节点
	nodes, err := r.repo.SelectKnowledgeTopicNodesByKbIds(ctx, kbIds)
	if err != nil {
		logx.Warnf("加载主题节点失败，意图路由降级为空结果: kbIds=%v, err=%v", kbIds, err)
		return nil
	}
	if len(nodes) == 0 {
		return nil
	}

	// 3. 渲染 userPrompt
	userPrompt, err := r.renderer.Render(intentScoreTemplateName, map[string]any{
		"question": question,
		"nodes":    nodes,
	})
	if err != nil {
		logx.Warnf("渲染主题意图提示词失败，意图路由降级为空结果: err=%v", err)
		return nil
	}

	// 4. 带超时调用大模型（不传 opts）
	scoreCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	raw, err := r.chatModel.Generate(scoreCtx, "", userPrompt)
	if err != nil {
		logx.Warnf("调用大模型进行主题意图打分失败，意图路由降级为空结果: err=%v", err)
		return nil
	}

	// 5. 容错解析输出（容忍 Markdown 围栏等包裹）
	var parsed intentScoreResult
	if err = utils.Unmarshal(raw, &parsed); err != nil {
		logx.Warnf("解析主题意图打分结果失败，意图路由降级为空结果: raw=%s, err=%v", raw, err)
		return nil
	}

	// 6. 以节点列表为准 join 分数：topicId -> 归一化后的最高分
	scoreMap := make(map[int64]float64, len(parsed.Scores))
	for _, item := range parsed.Scores {
		score := utils.Clamp(item.Score, 0, 1)
		// 重复 topicId 取最大
		if existed, ok := scoreMap[item.TopicId]; !ok || score > existed {
			scoreMap[item.TopicId] = score
		}
	}

	// 7. 输出顺序与 nodes 一致，长度与节点数相等，LLM 未返回的节点补 0
	results := make([]*NodeScore, 0, len(nodes))
	for _, node := range nodes {
		results = append(results, &NodeScore{
			Node:  node,
			Score: scoreMap[node.ID],
		})
	}
	return results
}
