package graph

import (
	"cmp"
	"slices"
	"strings"

	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
)

// Typed chunk（M3）：把图谱浓缩为"实体卡 / 关系卡 / 社区卡"，向量化后写入向量库，
// 使 vector 通道也能命中图谱内容（多跳关系、主题聚合语义）。
// 卡片为虚拟 DocumentChunk（负 ID + 文档ID），不写 MySQL、不参与父块提升。

// 卡片预算按 实体:关系:社区 = 2:5:3 分配
const (
	typedEntityShare    = 2
	typedRelationShare  = 5
	typedCommunityShare = 3
	typedShareTotal     = typedEntityShare + typedRelationShare + typedCommunityShare
)

// typedCardID 生成跨文档唯一的主键（负 int64 数字主键，避免与正数 chunk id 冲突）
func typedCardID(documentId, seq int64) int64 {
	return -(documentId*1_000_000 + seq)
}

// GenerateTypedCards 从图快照生成图卡片（按重要性截断，预算内确定性排序）。
// cardTopK <= 0 时不生成。
func GenerateTypedCards(snapshot *entity.GraphSnapshot, cardTopK int) []*entity.DocumentChunk {
	if snapshot == nil || cardTopK <= 0 {
		return nil
	}
	entityN := min(len(snapshot.Nodes), cardTopK*typedEntityShare/typedShareTotal)
	relN := min(len(snapshot.Edges), cardTopK*typedRelationShare/typedShareTotal)
	communityN := min(len(snapshot.Communities), cardTopK*typedCommunityShare/typedShareTotal)

	cards := make([]*entity.DocumentChunk, 0, entityN+relN+communityN)
	var seq int64
	push := func(text string) {
		seq++
		chunkText := utils.CompactWhitespace(text)
		if chunkText == "" {
			return
		}
		cards = append(cards, &entity.DocumentChunk{
			ID:         typedCardID(snapshot.DocumentId, seq),
			DocumentId: snapshot.DocumentId,
			TaskId:     snapshot.TaskId,
			PlanId:     snapshot.StrategyPlanId,
			ChunkType:  "graph_typed_card",
			ChunkText:  chunkText,
			CharCount:  utils.Len(chunkText),
			TokenCount: utils.EstimateTokens(chunkText),
		})
	}

	// 实体卡：按 rankBoost 降序截断
	entityNodes := byRankBoostNodes(snapshot.Nodes)
	for _, n := range utils.Limit(entityNodes, entityN) {
		aliases := utils.FilterBlank(n.Aliases)
		text := "[实体卡] " + n.Name
		if n.EntityType != "" {
			text += "（" + n.EntityType + "）"
		}
		if desc := strings.TrimSpace(n.Description); desc != "" && !strings.EqualFold(desc, n.Name) {
			text += "：" + desc
		}
		if len(aliases) > 0 {
			text += "。别名：" + strings.Join(utils.Limit(aliases, 3), "、")
		}
		push(text)
	}
	// 关系卡：按 rankBoost 降序截断
	relEdges := byRankBoostEdges(snapshot.Edges)
	for _, e := range utils.Limit(relEdges, relN) {
		head := nameOf(snapshot.Nodes, e.SourceKey)
		tail := nameOf(snapshot.Nodes, e.TargetKey)
		if head == "" || tail == "" {
			continue
		}
		text := "[关系卡] " + head + " " + e.RelationType + " " + tail
		if quote := utils.FirstNonBlank(utils.FirstNonBlank(e.QuoteTexts...), e.PredicateQuote, e.Description); quote != "" {
			text += "：" + quote
		}
		push(text)
	}
	// 社区卡：全量（已是重要序）
	for _, c := range utils.Limit(snapshot.Communities, communityN) {
		push("[社区卡] " + c.Title + "：" + c.Summary)
	}
	return cards
}

// byRankBoostNodes 按 rankBoost 降序（同分按 EntityKey）排序节点，返回新切片
func byRankBoostNodes(nodes []*entity.GraphNode) []*entity.GraphNode {
	out := utils.Copy(nodes)
	slices.SortFunc(out, func(a, b *entity.GraphNode) int {
		if c := cmp.Compare(b.RankBoost, a.RankBoost); c != 0 {
			return c
		}
		return cmp.Compare(a.EntityKey, b.EntityKey)
	})
	return out
}

func byRankBoostEdges(edges []*entity.GraphEdge) []*entity.GraphEdge {
	out := utils.Copy(edges)
	slices.SortFunc(out, func(a, b *entity.GraphEdge) int {
		if c := cmp.Compare(b.RankBoost, a.RankBoost); c != 0 {
			return c
		}
		if c := cmp.Compare(a.SourceKey, b.SourceKey); c != 0 {
			return c
		}
		if c := cmp.Compare(a.TargetKey, b.TargetKey); c != 0 {
			return c
		}
		return cmp.Compare(a.RelationType, b.RelationType)
	})
	return out
}

func nameOf(nodes []*entity.GraphNode, key string) string {
	for _, n := range nodes {
		if n != nil && n.EntityKey == key {
			return n.Name
		}
	}
	return ""
}
