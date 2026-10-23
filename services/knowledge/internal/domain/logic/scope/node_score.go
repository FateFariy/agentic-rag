package scope

import "github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"

// NodeScore topic 节点的意图打分结果
type NodeScore struct {
	Node  *entity.KnowledgeTopicNode // 被打分的主题节点
	Score float64                    // 意图相关性分数，取值区间 [0,1]，0 表示未命中或 LLM 未返回该节点
}
