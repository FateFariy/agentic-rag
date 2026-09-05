package community

// UnionFind 并查集，用于连通分量合并与去环辅助
type UnionFind struct {
	parent []int
}

// NewUnionFind 初始化大小为 n 的并查集
func NewUnionFind(n int) *UnionFind {
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	return &UnionFind{parent: parent}
}

// Find 路径压缩查找根
func (u *UnionFind) Find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

// Union 合并两个集合
func (u *UnionFind) Union(a, b int) {
	ra, rb := u.Find(a), u.Find(b)
	if ra != rb {
		u.parent[rb] = ra
	}
}

// Connected 判断两点是否连通
func (u *UnionFind) Connected(a, b int) bool {
	return u.Find(a) == u.Find(b)
}
