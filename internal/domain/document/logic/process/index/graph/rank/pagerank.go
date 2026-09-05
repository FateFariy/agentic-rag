package rank

import "github.com/swiftbit/know-agent/internal/domain/document/model/entity"

type outEdge struct {
	to int
	w  float64
}

// PageRank 带权 PageRank，返回 entity_key -> 重要性分数。
// 仅依赖入边权重，dangling 节点权重均匀分配。
func PageRank(nodes []*entity.GraphNode, edges []*entity.GraphEdge, damping float64, iterations int) map[string]float64 {
	n := len(nodes)
	rank := make(map[string]float64, n)
	if n == 0 {
		return rank
	}
	keyIdx := make(map[string]int, n)
	for i, nd := range nodes {
		keyIdx[nd.EntityKey] = i
		rank[nd.EntityKey] = 1.0 / float64(n)
	}
	adj := make([][]outEdge, n)
	outW := make([]float64, n)
	for _, e := range edges {
		s, ok1 := keyIdx[e.SourceKey]
		tg, ok2 := keyIdx[e.TargetKey]
		if !ok1 || !ok2 || s == tg {
			continue
		}
		w := e.Weight
		if w <= 0 {
			w = 1
		}
		adj[s] = append(adj[s], outEdge{tg, w})
		outW[s] += w
	}
	d := damping
	if d <= 0 || d >= 1 {
		d = 0.85
	}
	iters := iterations
	if iters <= 0 {
		iters = 20
	}
	for it := 0; it < iters; it++ {
		next := make(map[string]float64, n)
		for _, nd := range nodes {
			next[nd.EntityKey] = (1 - d) / float64(n)
		}
		var dangling float64
		for i, nd := range nodes {
			if outW[i] == 0 {
				dangling += rank[nd.EntityKey]
			}
		}
		danglingContrib := d * dangling / float64(n)
		for i := range adj {
			if outW[i] == 0 {
				continue
			}
			contrib := d * rank[nodes[i].EntityKey] / outW[i]
			for _, e := range adj[i] {
				next[nodes[e.to].EntityKey] += contrib
			}
		}
		for _, nd := range nodes {
			next[nd.EntityKey] += danglingContrib
		}
		rank = next
	}
	return rank
}
