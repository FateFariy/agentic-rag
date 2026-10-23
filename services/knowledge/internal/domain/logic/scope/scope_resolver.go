package scope

import (
	"cmp"
	"math"
	"slices"

	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

// RetrievalScopeResolver 检索范围解析器
type RetrievalScopeResolver struct{}

// NewRetrievalScopeResolver 创建检索范围解析器
func NewRetrievalScopeResolver() *RetrievalScopeResolver {
	return &RetrievalScopeResolver{}
}

// Resolve 解析收窄后的文档ID集合：
//   - 返回 nil 表示 GLOBAL（全库检索），消费方应回退到全部授权文档；
//   - 返回非空切片表示 DIRECTED（定向检索），内容为「命中绑定文档 ∩ 授权文档」
//     拼接「补充路」文档，整体去重且顺序稳定。
//
// 解析规则：
//  1. scopeConfig 为 nil、Validate 失败、scores 或 allowedDocs 为空 → GLOBAL；
//  2. topScore（各节点最高分）< ConfidenceThreshold → GLOBAL（等于阈值算 DIRECTED）；
//  3. 命中节点：Score 严格大于 MinIntentScore（==MinIntentScore 的节点忽略）；
//  4. 命中文档候选：topicBindings 中 TopicId 命中的 DocumentId 与 allowedDocs 取交集，
//     按 allowedDocs 的出现顺序输出并去重；
//  5. 交集为空（零绑定或绑定文档均未授权）→ GLOBAL；
//  6. 补充路：n=floor(SupplementRatio*len(allowedDocs))，上限为可用非命中授权数，
//     从 allowedDocs 按顺序取不在命中文档集合的前 n 个；
//  7. 返回命中文档后拼接补充文档（无重复、稳定顺序）。
func (r *RetrievalScopeResolver) Resolve(
	scores []*NodeScore,
	allowedDocs []int64,
	topicBindings []*entity.KnowledgeTopicDocumentRelation,
	scopeConfig *vo.ScopeConfig,
) []int64 {
	// 1. 防御性校验：配置非法或输入为空时回退全局检索
	if scopeConfig == nil {
		return nil
	}
	if err := scopeConfig.Validate(); err != nil {
		return nil
	}
	if len(scores) == 0 || len(allowedDocs) == 0 {
		return nil
	}

	// 2. 计算 topScore，低于置信阈值回退 GLOBAL（等于阈值进入 DIRECTED）
	topScoreNode := slices.MaxFunc(scores, func(a, b *NodeScore) int {
		return cmp.Compare(a.Score, b.Score)
	})
	if topScoreNode.Score < scopeConfig.ConfidenceThreshold {
		return nil
	}

	// 3. 收集命中节点（Score 必须严格大于 MinIntentScore）
	hitNodeIds := make(map[int64]struct{})
	for _, item := range scores {
		if item.Node == nil {
			continue
		}
		if item.Score > scopeConfig.MinIntentScore {
			hitNodeIds[item.Node.ID] = struct{}{}
		}
	}

	// 4. 收集命中节点绑定的文档候选集合
	hitDocSet := make(map[int64]struct{})
	for _, binding := range topicBindings {
		if binding == nil {
			continue
		}
		if _, ok := hitNodeIds[binding.TopicId]; ok {
			hitDocSet[binding.DocumentId] = struct{}{}
		}
	}

	// 4. 与授权文档取交集：按 allowedDocs 的出现顺序过滤并去重
	targetSet := make(map[int64]struct{})
	targetDocs := make([]int64, 0)
	for _, docId := range allowedDocs {
		if _, hit := hitDocSet[docId]; !hit {
			continue
		}
		targetSet[docId] = struct{}{}
		targetDocs = append(targetDocs, docId)
	}

	// 5. 交集为空 → GLOBAL
	if len(targetDocs) == 0 {
		return nil
	}

	// 6. 补充路：按比例从授权文档中顺序补齐非命中文档
	supplementCount := int(math.Floor(scopeConfig.SupplementRatio * float64(len(allowedDocs))))
	// 上限：可用的非命中授权文档数量
	supplementCount = min(len(allowedDocs)-len(targetDocs), supplementCount)
	supplementDocs := make([]int64, 0, supplementCount)
	for _, docId := range allowedDocs {
		if len(supplementDocs) >= supplementCount {
			break
		}
		// 命中文档不参与补充
		if _, inTarget := targetSet[docId]; inTarget {
			continue
		}
		supplementDocs = append(supplementDocs, docId)
	}

	// 7. 命中文档在前、补充文档在后，稳定顺序输出
	return append(targetDocs, supplementDocs...)
}
