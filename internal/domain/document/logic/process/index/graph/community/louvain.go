package community

import (
	"sort"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

type neighbor struct {
	to     int
	weight float64
}

// DetectLouvain 带权重的 Louvain 社区发现（单层局部移动 + 移除语义）。
// 返回与 nodes 等长的切片：下标 i 的节点所属社区编号。
func DetectLouvain(nodes []*entity.GraphNode, edges []*entity.GraphEdge, resolution float64) []int {
	n := len(nodes)
	if n == 0 {
		return nil
	}
	if resolution <= 0 {
		resolution = 1.0
	}
	keyIdx := make(map[string]int, n)
	for i, nd := range nodes {
		keyIdx[nd.EntityKey] = i
	}
	adj := make([][]neighbor, n)
	degrees := make([]float64, n)
	var totalWeight float64
	for _, e := range edges {
		si, ok1 := keyIdx[e.SourceKey]
		ti, ok2 := keyIdx[e.TargetKey]
		if !ok1 || !ok2 || si == ti {
			continue
		}
		w := e.Weight
		if w <= 0 {
			w = 1
		}
		adj[si] = append(adj[si], neighbor{ti, w})
		adj[ti] = append(adj[ti], neighbor{si, w})
		degrees[si] += w
		degrees[ti] += w
		totalWeight += 2 * w
	}
	if totalWeight == 0 {
		totalWeight = 1
	}
	comm := make([]int, n)
	commDeg := make([]float64, n)
	for i := range comm {
		comm[i] = i
		commDeg[i] = degrees[i]
	}
	improved := true
	for iter := 0; iter < 20 && improved; iter++ {
		improved = false
		for i := 0; i < n; i++ {
			ki := degrees[i]
			ci := comm[i]
			neighComm := make(map[int]float64)
			for _, e := range adj[i] {
				neighComm[comm[e.to]] += e.weight
			}
			// 标准 Louvain：先把 i 从 ci 移除再比较
			stayGain := neighComm[ci] - resolution*ki*(commDeg[ci]-ki)/totalWeight
			bestComm := ci
			bestGain := stayGain
			for c, wic := range neighComm {
				if c == ci {
					continue
				}
				gain := wic - resolution*ki*commDeg[c]/totalWeight
				if gain > bestGain {
					bestGain = gain
					bestComm = c
				}
			}
			if bestComm != ci {
				commDeg[ci] -= ki
				commDeg[bestComm] += ki
				comm[i] = bestComm
				improved = true
			}
		}
	}
	return comm
}

func nodeRank(nodes []*entity.GraphNode, ranks map[string]float64, idx int) float64 {
	if r, ok := ranks[nodes[idx].EntityKey]; ok {
		return r
	}
	return 0
}

// BuildCommunities 根据社区划分生成抽取式社区报告（标题取 top 实体名，零 LLM 成本）。
// CommunityNo 按社区重要性降序从 1 编号，作为 (document_id, task_id) 内唯一落库键。
func BuildCommunities(nodes []*entity.GraphNode, commIDs []int, ranks map[string]float64, topN int) []*entity.GraphCommunity {
	if len(nodes) == 0 || len(commIDs) != len(nodes) {
		return nil
	}
	if topN <= 0 {
		topN = 5
	}
	groups := make(map[int][]int)
	for i, c := range commIDs {
		groups[c] = append(groups[c], i)
	}
	communities := make([]*entity.GraphCommunity, 0, len(groups))
	for _, members := range groups {
		sort.Slice(members, func(a, b int) bool {
			return nodeRank(nodes, ranks, members[a]) > nodeRank(nodes, ranks, members[b])
		})
		names := make([]string, 0, len(members))
		memberKeys := make([]string, 0, len(members))
		var sum float64
		for _, mi := range members {
			names = append(names, nodes[mi].Name)
			memberKeys = append(memberKeys, nodes[mi].EntityKey)
			sum += nodeRank(nodes, ranks, mi)
		}
		titleParts := names
		if len(titleParts) > topN {
			titleParts = titleParts[:topN]
		}
		communities = append(communities, &entity.GraphCommunity{
			Title:       strings.Join(titleParts, " · "),
			Summary:     strings.Join(titleParts, "、") + " 等",
			MemberCount: len(names),
			MemberKeys:  memberKeys,
			RankBoost:   sum / float64(len(names)),
		})
	}
	sort.Slice(communities, func(i, j int) bool { return communities[i].RankBoost > communities[j].RankBoost })
	for i := range communities {
		communities[i].CommunityNo = i + 1
	}
	return communities
}
