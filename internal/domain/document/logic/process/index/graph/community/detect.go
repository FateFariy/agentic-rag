package community

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// minLouvainSplitSize 进入 Louvain 细化阶段的最小连通分量成员数
const minLouvainSplitSize = 3

// DetectUnionFind 仅并查集连通分量（保底模式，LouvainEnabled=false 时使用）：
// 每个连通分量一个社区，孤立点独立成社区，编号 0..k-1 确定性压缩。
func DetectUnionFind(nodes []*entity.GraphNode, edges []*entity.GraphEdge) []int {
	n := len(nodes)
	if n == 0 {
		return nil
	}
	keyIdx := utils.IndexBy(nodes, func(nd *entity.GraphNode) string { return nd.EntityKey })
	uf := NewUnionFind(n)
	for _, e := range edges {
		s, ok1 := keyIdx[e.SourceKey]
		t, ok2 := keyIdx[e.TargetKey]
		if ok1 && ok2 && s != t {
			uf.Union(s, t)
		}
	}
	comm := make([]int, n)
	labelOf := make(map[int]int)
	next := 0
	for i := 0; i < n; i++ {
		root := uf.Find(i)
		label, ok := labelOf[root]
		if !ok {
			label = next
			labelOf[root] = label
			next++
		}
		comm[i] = label
	}
	return comm
}

// DetectCommunities 两阶段社区发现（M1 保底 + 细化）：
//  阶段一：并查集连通分量（保底，任何图都给出全量归属，孤立点独立成社区，不 panic）
//  阶段二：对成员数不少于 minLouvainSplitSize 的连通分量，在其内部运行加权 Louvain 细化
// 返回与 nodes 等长切片：节点 -> 社区编号（0..k-1，确定性编号）。
func DetectCommunities(nodes []*entity.GraphNode, edges []*entity.GraphEdge, resolution float64) []int {
	n := len(nodes)
	if n == 0 {
		return nil
	}
	if resolution <= 0 {
		resolution = 1.0
	}
	keyIdx := utils.IndexBy(nodes, func(nd *entity.GraphNode) string { return nd.EntityKey })
	uf := NewUnionFind(n)
	for _, e := range edges {
		s, ok1 := keyIdx[e.SourceKey]
		t, ok2 := keyIdx[e.TargetKey]
		if ok1 && ok2 && s != t {
			uf.Union(s, t)
		}
	}

	comm := make([]int, n)
	for i := range comm {
		comm[i] = -1
	}
	visited := make([]bool, n)
	nextID := 0
	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}
		root := uf.Find(i)
		var members []int
		for j := 0; j < n; j++ {
			if uf.Find(j) == root {
				members = append(members, j)
			}
		}
		for _, j := range members {
			visited[j] = true
		}
		if len(members) < minLouvainSplitSize {
			for _, j := range members {
				comm[j] = nextID
			}
			nextID++
			continue
		}
		// 分量内部 Louvain 细化（Louvain 天然不跨连通分量移动）
		local := runLouvainOnSubset(nodes, edges, members, resolution)
		if countCommunities(local) <= 1 {
			for _, j := range members {
				comm[j] = nextID
			}
			nextID++
			continue
		}
		for li, j := range members {
			comm[j] = nextID + local[li]
		}
		nextID += countCommunities(local)
	}
	return compressLabels(comm)
}

// runLouvainOnSubset 在指定节点子集构成的诱导子图上运行 Louvain，
// 返回与 members 等长的局部社区编号切片。
func runLouvainOnSubset(nodes []*entity.GraphNode, edges []*entity.GraphEdge, members []int, resolution float64) []int {
	subNodes := utils.Map(members, func(j int) *entity.GraphNode { return nodes[j] })
	subKeys := utils.IndexBy(subNodes, func(nd *entity.GraphNode) string { return nd.EntityKey })
	subEdges := utils.Filter(edges, func(e *entity.GraphEdge) bool {
		_, sIn := subKeys[e.SourceKey]
		_, tIn := subKeys[e.TargetKey]
		return sIn && tIn
	})
	return DetectLouvain(subNodes, subEdges, resolution)
}

func countCommunities(comm []int) int {
	return len(utils.Distinct(comm, func(c int) int { return c }))
}

// compressLabels 将社区编号压缩为 0..k-1（按首次出现顺序，保证确定性）
func compressLabels(comm []int) []int {
	if len(comm) == 0 {
		return comm
	}
	m := make(map[int]int, len(comm))
	next := 0
	for i, c := range comm {
		v, ok := m[c]
		if !ok {
			v = next
			m[c] = v
			next++
		}
		comm[i] = v
	}
	return comm
}

// Modularity 计算加权无向图的模块度（新曼-吉文），用于质量可观测与分区择优。
// 返回 0 表示无有效边或划分非法。
func Modularity(nodes []*entity.GraphNode, edges []*entity.GraphEdge, commIDs []int) float64 {
	if len(nodes) == 0 || len(commIDs) != len(nodes) {
		return 0
	}
	keyIdx := utils.IndexBy(nodes, func(nd *entity.GraphNode) string { return nd.EntityKey })
	var totalW float64
	commDeg := make(map[int]float64)
	commInternal := make(map[int]float64)
	for _, e := range edges {
		s, ok1 := keyIdx[e.SourceKey]
		t, ok2 := keyIdx[e.TargetKey]
		if !ok1 || !ok2 || s == t {
			continue
		}
		w := e.Weight
		if w <= 0 {
			w = 1
		}
		cs, ct := commIDs[s], commIDs[t]
		commDeg[cs] += w
		commDeg[ct] += w
		totalW += w
		if cs == ct {
			commInternal[cs] += w
		}
	}
	if totalW <= 0 {
		return 0
	}
	var q float64
	for c := range commDeg {
		frac := commDeg[c] / (2 * totalW)
		q += commInternal[c]/totalW - frac*frac
	}
	return q
}
