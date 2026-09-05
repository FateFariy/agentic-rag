package merge

import (
	"hash/fnv"
	"sort"
	"strconv"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/extract"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// GraphData 解析/合并后的图谱数据
type GraphData struct {
	Nodes []*entity.GraphNode
	Edges []*entity.GraphEdge
}

// Resolver 实体规范化与关系合并（消解键 = entity_key）
type Resolver interface {
	Resolve(raws []*extract.RawGraph) *GraphData
}

type graphResolver struct{}

// NewResolver 创建合并解析器
func NewResolver() Resolver {
	return &graphResolver{}
}

// normalizeWidth 全角字符转半角（ASCII 范围 + 全角空格）
func normalizeWidth(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r == '\u3000':
			runes[i] = ' '
		case r >= '\uff01' && r <= '\uff5e':
			runes[i] = r - 0xfee0
		}
	}
	return string(runes)
}

// canonical 实体名规范化：去首尾空格 + 全角转半角 + ASCII 大写，作为 normalized_name
func canonical(name string) string {
	return strings.ToUpper(normalizeWidth(strings.TrimSpace(name)))
}

// normalizeEntityType 实体类型受控归一：白名单内原样返回，未命中归入"概念"防类型爆炸
func normalizeEntityType(t string) string {
	if t = strings.TrimSpace(t); enum.IsValidGraphEntityType(t) {
		return t
	}
	return "概念"
}

// mergeKey 内存消解键：同 normalized_name + 同受控 type 视为同一实体
func mergeKey(normalizedName, entityType string) string {
	return normalizedName + "\x1f" + entityType
}

// entityKey 计算 entity_key = fnv1a64(normalized_name + "\x1f" + entity_type)，首字母 E 保证落库键可读性
func entityKey(normalizedName, entityType string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(mergeKey(normalizedName, entityType)))
	return "E" + strconv.FormatUint(h.Sum64(), 36)
}

type entAcc struct {
	node       *entity.GraphNode
	nameSet    map[string]struct{}
	mentionCnt int
}

type edgeAcc struct {
	edge     *entity.GraphEdge
	chunkSet map[int64]struct{}
	quoteSet map[string]struct{}
}

type edgeKey struct {
	src, tgt, rel string
}

// Resolve 两轮完成实体消解与关系合并：
// 第一轮收集实体并归一化（全半角/大小写/别名累积/mention 计数）；
// 第二轮合并关系（源/目标必须是已抽实体，自环丢弃，同 (src,tgt,type) 权重累加）。
func (r *graphResolver) Resolve(raws []*extract.RawGraph) *GraphData {
	entMap := make(map[string]*entAcc)
	canonToKeys := make(map[string][]string)

	for _, raw := range raws {
		if raw == nil {
			continue
		}
		for _, e := range raw.Entities {
			normalized := canonical(e.Name)
			if normalized == "" {
				continue
			}
			etype := normalizeEntityType(e.Type)
			key := mergeKey(normalized, etype)
			acc, ok := entMap[key]
			if !ok {
				acc = &entAcc{
					node: &entity.GraphNode{
						EntityKey:      entityKey(normalized, etype),
						NormalizedName: normalized,
						EntityType:     etype,
						Name:           strings.TrimSpace(e.Name),
					},
					nameSet: map[string]struct{}{},
				}
				entMap[key] = acc
				canonToKeys[normalized] = append(canonToKeys[normalized], key)
			}
			acc.mentionCnt++
			if name := strings.TrimSpace(e.Name); name != "" {
				acc.nameSet[name] = struct{}{}
			}
			if len([]rune(e.Description)) > len([]rune(acc.node.Description)) {
				acc.node.Description = strings.TrimSpace(e.Description)
			}
		}
	}

	keys := make([]string, 0, len(entMap))
	for k := range entMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	nodes := make([]*entity.GraphNode, 0, len(keys))
	for _, k := range keys {
		acc := entMap[k]
		acc.node.MentionCount = acc.mentionCnt
		acc.node.Aliases = sortedSet(acc.nameSet)
		if longest := longestName(acc.node.Aliases); longest != "" {
			acc.node.Name = longest
		}
		nodes = append(nodes, acc.node)
	}

	edgeMap := make(map[edgeKey]*edgeAcc)
	for _, raw := range raws {
		if raw == nil {
			continue
		}
		for _, rel := range raw.Relations {
			srcKey := firstEntityKey(canonToKeys, entMap, canonical(rel.Source))
			tgtKey := firstEntityKey(canonToKeys, entMap, canonical(rel.Target))
			if srcKey == "" || tgtKey == "" || srcKey == tgtKey {
				continue
			}
			relType := enum.NormalizeRelationType(strings.TrimSpace(rel.RelationType))
			ek := edgeKey{srcKey, tgtKey, relType}
			acc, ok := edgeMap[ek]
			if !ok {
				acc = &edgeAcc{
					edge: &entity.GraphEdge{
						SourceKey:      srcKey,
						TargetKey:      tgtKey,
						RelationType:   relType,
						PredicateQuote: strings.TrimSpace(rel.Quote),
					},
					chunkSet: map[int64]struct{}{},
					quoteSet: map[string]struct{}{},
				}
				edgeMap[ek] = acc
			}
			acc.edge.Weight++
			if len([]rune(rel.Description)) > len([]rune(acc.edge.Description)) {
				acc.edge.Description = strings.TrimSpace(rel.Description)
			}
			if rel.SourceChunkId > 0 {
				acc.chunkSet[rel.SourceChunkId] = struct{}{}
			}
			if q := strings.TrimSpace(rel.Quote); q != "" {
				acc.quoteSet[q] = struct{}{}
			}
		}
	}

	edgeKeys := make([]edgeKey, 0, len(edgeMap))
	for ek := range edgeMap {
		edgeKeys = append(edgeKeys, ek)
	}
	sort.Slice(edgeKeys, func(i, j int) bool {
		if edgeKeys[i].src != edgeKeys[j].src {
			return edgeKeys[i].src < edgeKeys[j].src
		}
		if edgeKeys[i].tgt != edgeKeys[j].tgt {
			return edgeKeys[i].tgt < edgeKeys[j].tgt
		}
		return edgeKeys[i].rel < edgeKeys[j].rel
	})
	edges := make([]*entity.GraphEdge, 0, len(edgeKeys))
	for _, ek := range edgeKeys {
		acc := edgeMap[ek]
		chunkIDs := make([]int64, 0, len(acc.chunkSet))
		for id := range acc.chunkSet {
			chunkIDs = append(chunkIDs, id)
		}
		sort.Slice(chunkIDs, func(i, j int) bool { return chunkIDs[i] < chunkIDs[j] })
		acc.edge.ChunkIds = chunkIDs
		acc.edge.QuoteTexts = sortedSet(acc.quoteSet)
		edges = append(edges, acc.edge)
	}
	return &GraphData{Nodes: nodes, Edges: edges}
}

// firstEntityKey 按规范化名匹配候选实体并取第一个（canonToKeys 按键确定性排序）
func firstEntityKey(canonToKeys map[string][]string, entMap map[string]*entAcc, normalized string) string {
	keys := canonToKeys[normalized]
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	return entMap[keys[0]].node.EntityKey
}

func sortedSet(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func longestName(names []string) string {
	var longest string
	for _, n := range names {
		if len([]rune(n)) > len([]rune(longest)) {
			longest = n
		}
	}
	return longest
}
