package graph

import (
	"context"
	"fmt"
	"sort"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/community"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/extract"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/merge"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/rank"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// Builder 实现 index.GraphRagBuilder：编排 抽取→消解合并→成本闸门→PageRank→社区发现→事务化全删全插（M1）
type Builder struct {
	store     adapter.GraphStore
	extractor Extractor
	resolver  merge.Resolver
	config    *config.GraphRagConf
}

// NewBuildService 创建图谱构建服务
func NewBuildService(store adapter.GraphStore, extractor Extractor, resolver merge.Resolver, cfg *config.GraphRagConf) *Builder {
	return &Builder{
		store:     store,
		extractor: extractor,
		resolver:  resolver,
		config:    cfg,
	}
}

// RebuildDocumentGraph 构建并落库单篇文档图谱。
// 任一环节失败均返回 error，由 GraphRagStage 记录失败结果并按 ErrGraphRagBuildFailed 降级，
// 保证「图谱失败不影响索引任务完成」。
func (s *Builder) RebuildDocumentGraph(ctx context.Context, documentId, taskId, knowledgeBaseId int64, chunks []*entity.DocumentChunk) (*vo.GraphRagBuildResult, error) {
	result := &vo.GraphRagBuildResult{}
	if len(chunks) == 0 {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeEmpty
		result.GraphPersistenceReason = "没有可构建图谱的文档块"
		return result, nil
	}
	// 1. 逐块抽取（LLM 调用失败仅告警跳过该块，保留其余）
	var raws []*extract.RawGraph
	droppedEntity, droppedRelation := 0, 0
	for _, ch := range chunks {
		rg, err := s.extractor.Extract(ctx, ch)
		if err != nil {
			logx.Warnf("抽取文档块图谱失败 chunkId=%d: %v", ch.ID, err)
			continue
		}
		if rg != nil {
			raws = append(raws, rg)
			droppedEntity += rg.DroppedEntities
			droppedRelation += rg.DroppedRelations
		}
	}
	if droppedEntity > 0 || droppedRelation > 0 {
		logx.Infof("grounding 校验丢弃不可信抽取: documentId=%d, 实体=%d, 关系=%d", documentId, droppedEntity, droppedRelation)
	}
	// 2. 实体消解 + 关系合并
	data := s.resolver.Resolve(raws)
	if len(data.Nodes) == 0 {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeEmpty
		result.GraphPersistenceReason = "没有抽取到实体"
		return result, nil
	}
	// 3. PageRank（全图重要性，作为成本闸门截断依据）
	ranks := rank.PageRank(data.Nodes, data.Edges, s.config.PageRankDamping, s.config.PageRankMaxIter)
	// 4. 成本闸门：超限按重要性截断，避免 LLM 费用与图谱体积失控
	nodes, edges := enforceLimits(data.Nodes, data.Edges, s.config.MaxEntities, s.config.MaxRelations, documentId)
	if len(nodes) == 0 {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeEmpty
		result.GraphPersistenceReason = "成本闸门截断后无剩余实体"
		return result, nil
	}
	writeBackGraphMetrics(nodes, edges, ranks)
	// 5. 社区发现（并查集保底 + 可选 Louvain 细化）+ 抽取式社区报告
	var commIDs []int
	if s.config.LouvainEnabled {
		commIDs = community.DetectCommunities(nodes, edges, s.config.LouvainGamma)
	} else {
		commIDs = community.DetectUnionFind(nodes, edges)
	}
	communities := community.BuildCommunities(nodes, commIDs, ranks, s.config.CommunityTopNodes)
	communities = enforceCommunityLimit(communities, s.config.MaxCommunities, documentId)
	if mod := community.Modularity(nodes, edges, commIDs); mod > 0 {
		logx.Infof("图谱社区检测完成: documentId=%d, communities=%d, modularity=%.4f",
			documentId, len(communities), mod)
	}
	// 6. 无有效关系：仅实体入库（降级）
	if len(edges) == 0 {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeDegraded
		result.GraphPersistenceReason = "无有效关系，仅实体入库"
	}
	// 7. 事务化全删全插（幂等）
	snapshot := &entity.GraphSnapshot{
		DocumentId:      documentId,
		TaskId:          taskId,
		KnowledgeBaseId: knowledgeBaseId,
		Nodes:           nodes,
		Edges:           edges,
		Communities:     communities,
	}
	if err := s.store.ReplaceDocumentGraph(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("落库图谱失败: %w", err)
	}
	// 8. 结果统计
	result.EntityCount = len(nodes)
	result.RelationCount = len(edges)
	result.EvidenceCount = countEvidence(edges)
	result.CommunityCount = len(communities)
	if result.GraphPersistenceOutcome == "" {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeSuccess
	}
	result.KgCommitted = true
	return result, nil
}

// enforceLimits 成本闸门：实体/关系超上限时按 PageRank 降序 + entity_key 升序全序截断，
// 并同步过滤端点被裁剪的关系。
func enforceLimits(nodes []*entity.GraphNode, edges []*entity.GraphEdge, maxEntities, maxRelations int, documentId int64) ([]*entity.GraphNode, []*entity.GraphEdge) {
	if maxEntities > 0 && len(nodes) > maxEntities {
		ranked := make([]*entity.GraphNode, len(nodes))
		copy(ranked, nodes)
		sort.Slice(ranked, func(i, j int) bool {
			if ranked[i].PageRank != ranked[j].PageRank {
				return ranked[i].PageRank > ranked[j].PageRank
			}
			return ranked[i].EntityKey < ranked[j].EntityKey
		})
		kept := ranked[:maxEntities]
		keySet := make(map[string]struct{}, len(kept))
		for _, n := range kept {
			keySet[n.EntityKey] = struct{}{}
		}
		edges = filterEdgesByKeys(edges, keySet)
		nodes = kept
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].EntityKey < nodes[j].EntityKey })
		logx.Warnf("实体成本闸门截断: documentId=%d, 保留 %d/%d", documentId, len(nodes), maxEntities)
	}
	if maxRelations > 0 && len(edges) > maxRelations {
		ranked := make([]*entity.GraphEdge, len(edges))
		copy(ranked, edges)
		sort.Slice(ranked, func(i, j int) bool {
			if ranked[i].RankBoost != ranked[j].RankBoost {
				return ranked[i].RankBoost > ranked[j].RankBoost
			}
			if ranked[i].SourceKey != ranked[j].SourceKey {
				return ranked[i].SourceKey < ranked[j].SourceKey
			}
			if ranked[i].TargetKey != ranked[j].TargetKey {
				return ranked[i].TargetKey < ranked[j].TargetKey
			}
			return ranked[i].RelationType < ranked[j].RelationType
		})
		edges = ranked[:maxRelations]
		logx.Warnf("关系成本闸门截断: documentId=%d, 保留 %d/%d", documentId, len(edges), maxRelations)
	}
	return nodes, edges
}

// filterEdgesByKeys 仅保留两端实体都在保留集合内的关系（成本闸门裁剪实体后同步裁剪关系）
func filterEdgesByKeys(edges []*entity.GraphEdge, keySet map[string]struct{}) []*entity.GraphEdge {
	return utils.Filter(edges, func(e *entity.GraphEdge) bool {
		_, sOK := keySet[e.SourceKey]
		_, tOK := keySet[e.TargetKey]
		return sOK && tOK
	})
}

func enforceCommunityLimit(communities []*entity.GraphCommunity, maxCommunities int, documentId int64) []*entity.GraphCommunity {
	if maxCommunities <= 0 || len(communities) <= maxCommunities {
		return communities
	}
	// BuildCommunities 已按 RankBoost 降序，直接截断前 N 个即可
	communities = communities[:maxCommunities]
	logx.Warnf("社区成本闸门截断: documentId=%d, 保留 %d 个社区", documentId, len(communities))
	return communities
}

// writeBackGraphMetrics 写回算法指标：节点 pagerank/rank_boost/degree，关系 rank_boost=两端均值
func writeBackGraphMetrics(nodes []*entity.GraphNode, edges []*entity.GraphEdge, ranks map[string]float64) {
	byKey := make(map[string]*entity.GraphNode, len(nodes))
	for _, n := range nodes {
		n.PageRank = ranks[n.EntityKey]
		n.RankBoost = ranks[n.EntityKey]
		n.Degree = 0
		byKey[n.EntityKey] = n
	}
	for _, e := range edges {
		if s, ok := byKey[e.SourceKey]; ok {
			s.Degree++
		}
		if t, ok := byKey[e.TargetKey]; ok {
			t.Degree++
		}
		s, sOK := byKey[e.SourceKey]
		t, tOK := byKey[e.TargetKey]
		if sOK && tOK {
			e.RankBoost = (s.PageRank + t.PageRank) / 2
		}
	}
}

func countEvidence(edges []*entity.GraphEdge) int {
	return utils.Reduce(edges, func(n int, e *entity.GraphEdge) int { return n + len(e.ChunkIds) }, 0)
}
