package graph

import (
	"cmp"
	"hash/fnv"
	"slices"
	"strconv"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/extract"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// GraphData 解析/合并后的图谱数据
type GraphData struct {
	Nodes []*entity.GraphNode
	Edges []*entity.GraphEdge
}

type Resolver struct{}

// canonical 实体名规范化：去首尾空格 + 全角转半角 + ASCII 大写，作为 normalized_name
func canonical(name string) string {
	return strings.ToUpper(utils.ToHalfWidth(strings.TrimSpace(name)))
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
func (r *Resolver) Resolve(raws []*extract.RawGraph) *GraphData {
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
			etype := enum.NormalizeEntityType(e.Type)
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

	keys := utils.MapKeys(entMap)
	slices.Sort(keys)
	nodes := make([]*entity.GraphNode, 0, len(keys))
	for _, k := range keys {
		acc := entMap[k]
		acc.node.MentionCount = acc.mentionCnt
		acc.node.Aliases = utils.SortedKeys(acc.nameSet)
		longest := slices.MaxFunc(acc.node.Aliases, func(a, b string) int { return cmp.Compare(utils.Len(a), utils.Len(b)) })
		if longest != "" {
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

	edgeKeys := utils.MapKeys(edgeMap)
	slices.SortFunc(edgeKeys, func(a, b edgeKey) int {
		if c := cmp.Compare(a.src, b.src); c != 0 {
			return c
		}
		if c := cmp.Compare(a.tgt, b.tgt); c != 0 {
			return c
		}
		return cmp.Compare(a.rel, b.rel)
	})
	edges := make([]*entity.GraphEdge, 0, len(edgeKeys))
	for _, ek := range edgeKeys {
		acc := edgeMap[ek]
		chunkIds := utils.MapKeys(acc.chunkSet)
		slices.Sort(chunkIds)
		acc.edge.ChunkIds = chunkIds
		acc.edge.QuoteTexts = utils.SortedKeys(acc.quoteSet)
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
	slices.Sort(keys)
	return entMap[keys[0]].node.EntityKey
}
