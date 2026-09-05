package graph

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/common/logx"
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

// Builder 实现 index.GraphRagBuilder：编排 抽取→消解合并→PageRank→社区发现→事务化全删全插（M1）
type Builder struct {
	store     adapter.GraphStore
	extractor extract.Extractor
	resolver  merge.Resolver
	config    *config.GraphRagConf
}

// NewBuildService 创建图谱构建服务
func NewBuildService(store adapter.GraphStore, extractor extract.Extractor, resolver merge.Resolver, cfg *config.GraphRagConf) *Builder {
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
	// 3. PageRank 并写回节点 pagerank/rank_boost/degree 与关系 rank_boost
	ranks := rank.PageRank(data.Nodes, data.Edges, s.config.PageRankDamping, s.config.PageRankIterations)
	writeBackGraphMetrics(data, ranks)
	// 4. 社区发现（并查集连通分量保底 + 大分量内 Louvain 细化）+ 抽取式社区报告
	commIDs := community.DetectCommunities(data.Nodes, data.Edges, s.config.LouvainResolution)
	communities := community.BuildCommunities(data.Nodes, commIDs, ranks, s.config.CommunityTopNodes)
	if mod := community.Modularity(data.Nodes, data.Edges, commIDs); mod > 0 {
		logx.Infof("图谱社区检测完成: documentId=%d, communities=%d, modularity=%.4f",
			documentId, len(communities), mod)
	}
	// 5. 无有效关系：仅实体入库（降级）
	if len(data.Edges) == 0 {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeDegraded
		result.GraphPersistenceReason = "无有效关系，仅实体入库"
	}
	// 6. 事务化全删全插（幂等）
	snapshot := &entity.GraphSnapshot{
		DocumentId:      documentId,
		TaskId:          taskId,
		KnowledgeBaseId: knowledgeBaseId,
		Nodes:           data.Nodes,
		Edges:           data.Edges,
		Communities:     communities,
	}
	if err := s.store.CommitGraph(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("落库图谱失败: %w", err)
	}
	// 7. 结果统计
	result.EntityCount = len(data.Nodes)
	result.RelationCount = len(data.Edges)
	result.EvidenceCount = countEvidence(data.Edges)
	result.CommunityCount = len(communities)
	if result.GraphPersistenceOutcome == "" {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeSuccess
	}
	result.KgCommitted = true
	return result, nil
}

// writeBackGraphMetrics 写回算法指标：节点 pagerank/rank_boost/degree，关系 rank_boost=两端均值
func writeBackGraphMetrics(data *merge.GraphData, ranks map[string]float64) {
	byKey := make(map[string]*entity.GraphNode, len(data.Nodes))
	for _, n := range data.Nodes {
		n.PageRank = ranks[n.EntityKey]
		n.RankBoost = ranks[n.EntityKey]
		byKey[n.EntityKey] = n
	}
	for _, e := range data.Edges {
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
	var n int
	for _, e := range edges {
		n += len(e.ChunkIds)
	}
	return n
}
