package storage

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// ===================== 领域图模型 → Cypher 参数 =====================

func cloneParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func toEntityParams(nodes []*entity.GraphNode) []map[string]any {
	params := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		aliases := n.Aliases
		if aliases == nil {
			aliases = []string{}
		}
		params = append(params, map[string]any{
			"entity_key":      n.EntityKey,
			"name":            n.Name,
			"normalized_name": n.NormalizedName,
			"entity_type":     n.EntityType,
			"description":     n.Description,
			"aliases":         aliases,
			"mention_count":   n.MentionCount,
			"pagerank":        n.PageRank,
			"rank_boost":      n.RankBoost,
			"degree":          n.Degree,
		})
	}
	return params
}

func toCommunityParams(communities []*entity.GraphCommunity) []map[string]any {
	params := make([]map[string]any, 0, len(communities))
	for _, c := range communities {
		params = append(params, map[string]any{
			"community_no": c.CommunityNo,
			"level":        c.Level,
			"title":        c.Title,
			"summary":      c.Summary,
			"rank_boost":   c.RankBoost,
			"modularity":   c.Modularity,
			"member_count": c.MemberCount,
		})
	}
	return params
}

func toRelParams(edges []*entity.GraphEdge) []map[string]any {
	params := make([]map[string]any, 0, len(edges))
	for _, e := range edges {
		chunkIDs := e.ChunkIds
		if chunkIDs == nil {
			chunkIDs = []int64{}
		}
		quoteTexts := e.QuoteTexts
		if quoteTexts == nil {
			quoteTexts = []string{}
		}
		params = append(params, map[string]any{
			"source_key":      e.SourceKey,
			"target_key":      e.TargetKey,
			"relation_type":   utils.BlankToDefault(e.RelationType, "RELATED_TO"),
			"weight":          e.Weight,
			"rank_boost":      e.RankBoost,
			"predicate_quote": e.PredicateQuote,
			"description":     e.Description,
			"chunk_ids":       chunkIDs,
			"quote_texts":     quoteTexts,
		})
	}
	return params
}

// relationGroup 按受控关系类型分组的关系集合
type relationGroup struct {
	relationType string
	edges        []*entity.GraphEdge
}

// groupRelationsByType 将关系按 RelationType 分组，保证每种类型用一条静态 Cypher 落库
func groupRelationsByType(edges []*entity.GraphEdge) []relationGroup {
	groups := utils.GroupBy(edges, func(e *entity.GraphEdge) (string, *entity.GraphEdge) {
		return utils.BlankToDefault(e.RelationType, "RELATED_TO"), e
	})
	out := make([]relationGroup, 0, len(groups))
	for t, es := range groups {
		out = append(out, relationGroup{relationType: t, edges: es})
	}
	return out
}

func flattenCommunityMembers(communities []*entity.GraphCommunity) []map[string]any {
	var members []map[string]any
	for _, c := range communities {
		for _, key := range c.MemberKeys {
			members = append(members, map[string]any{
				"community_no": c.CommunityNo,
				"entity_key":   key,
			})
		}
	}
	return members
}
