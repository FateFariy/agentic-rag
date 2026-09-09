package graph

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/community"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index/graph/extract"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// GraphTypeProvider 提供知识库自定义的图谱实体/关系类型白名单。
// 由调用方注入（通常包装 KnowledgeRepository），避免 document 域直接依赖 knowledge 域。
type GraphTypeProvider interface {
	// GraphTypes 返回指定知识库允许的实体类型与关系类型；缺省返回 nil，调用方回退到全局默认白名单。
	GraphTypes(ctx context.Context, knowledgeBaseId int64) (entityTypes, relationTypes []string, err error)
}

// Builder 实现 index.GraphRagBuilder：编排 抽取→消解合并→成本闸门→PageRank→社区发现→LLM 社区报告→事务化全删全插→图卡片投影（M1+M3）
type Builder struct {
	store         adapter.GraphStore
	extractor     Extractor
	resolver      Resolver
	config        *config.GraphRagConf
	reporter      CommunityReporter     // M3：可选 LLM 社区报告器（nil 时使用抽取式摘要）
	vectorIndexer adapter.VectorIndexer // M3：可选图卡片向量化投影（nil 时关闭）
	typesProvider GraphTypeProvider     // 按知识库动态提供图谱类型白名单（nil 时回退全局默认）
}

// NewBuilder 创建图谱构建器
func NewBuilder(store adapter.GraphStore, extractor Extractor, cfg *config.GraphRagConf, reporter CommunityReporter, vectorIndexer adapter.VectorIndexer, typesProvider GraphTypeProvider) *Builder {
	return &Builder{
		store:         store,
		extractor:     extractor,
		resolver:      Resolver{},
		config:        cfg,
		reporter:      reporter,
		vectorIndexer: vectorIndexer,
		typesProvider: typesProvider,
	}
}

// RebuildDocumentGraph 构建并落库单篇文档图谱。
// 任一环节失败均返回 error，由 GraphRagStage 记录失败结果并按 ErrGraphRagBuildFailed 降级，保证「图谱失败不影响索引任务完成」
func (s *Builder) RebuildDocumentGraph(ctx context.Context, documentId, taskId, knowledgeBaseId int64, chunks []*entity.DocumentChunk) (*vo.GraphRagBuildResult, error) {
	result := &vo.GraphRagBuildResult{}
	if len(chunks) == 0 {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeEmpty
		result.GraphPersistenceReason = "没有可构建图谱的文档块"
		return result, nil
	}
	// 1. 并发分批抽取（LLM 调用失败仅告警跳过该块，保留其余）
	entityTypes, relationTypes := s.resolveGraphTypes(ctx, knowledgeBaseId)
	keyFunc := func(s string) (string, struct{}) { return s, struct{}{} }
	entityWhitelist := utils.MapBy(entityTypes, keyFunc)
	relationWhitelist := utils.MapBy(relationTypes, keyFunc)
	raws := s.extractAll(ctx, chunks, entityTypes, relationTypes)
	droppedEntity, droppedRelation := 0, 0
	for _, rg := range raws {
		droppedEntity += rg.DroppedEntities
		droppedRelation += rg.DroppedRelations
	}
	if droppedEntity > 0 || droppedRelation > 0 {
		logx.Infof("grounding 校验丢弃不可信抽取: documentId=%d, 实体=%d, 关系=%d", documentId, droppedEntity, droppedRelation)
	}
	// 2. 实体消解 + 关系合并
	data := s.resolver.Resolve(raws, entityWhitelist, relationWhitelist)
	if len(data.Nodes) == 0 {
		result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeEmpty
		result.GraphPersistenceReason = "没有抽取到实体"
		return result, nil
	}
	// 3. PageRank（全图重要性，作为成本闸门截断依据）
	ranks := PageRank(data.Nodes, data.Edges, s.config.PageRankDamping, s.config.PageRankMaxIter)
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
	communities = utils.Limit(communities, s.config.MaxCommunities)
	if mod := community.Modularity(nodes, edges, commIDs); mod > 0 {
		logx.Infof("图谱社区检测完成: documentId=%d, communities=%d, modularity=%.4f",
			documentId, len(communities), mod)
	}
	// 5.1 M3：LLM 社区报告（预算闸门：仅前 CommunityReportTopN 个重要社区；失败回退抽取式摘要）
	s.enhanceCommunitySummaries(ctx, nodes, communities)
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
	// 7.1 M3：Typed chunk 投影（图卡片写入向量库，增强依赖：失败仅降级告警，不阻断索引任务）
	s.projectTypedChunks(ctx, snapshot)
	// 8. 结果统计
	result.EntityCount = len(nodes)
	result.RelationCount = len(edges)
	result.EvidenceCount = utils.Reduce(edges, func(n int, e *entity.GraphEdge) int { return n + len(e.ChunkIds) }, 0)
	result.CommunityCount = len(communities)
	result.GraphPersistenceOutcome = enum.GraphPersistenceOutcomeSuccess
	result.KgCommitted = true
	return result, nil
}

// extractAll 并发分批抽取：ExtractBatch 控制每批 chunk 数，ExtractConcurrency 控制并发批次数。
// 单块抽取失败仅告警跳过；结果按 chunk 顺序归集，保证下游消解与图算法结果稳定。
func (s *Builder) extractAll(ctx context.Context, chunks []*entity.DocumentChunk, entityTypes, relationTypes []string) []*extract.RawGraph {
	concurrency := max(s.config.ExtractConcurrency, 1)
	batchSize := s.config.ExtractBatch
	if batchSize <= 0 {
		batchSize = len(chunks)
	}

	raws := make([]*extract.RawGraph, len(chunks))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for start := 0; start < len(chunks); start += batchSize {
		end := min(start+batchSize, len(chunks))
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return utils.Filter(raws, func(rg *extract.RawGraph) bool { return rg != nil })
		}
		wg.Add(1)
		go func(offset int, batch []*entity.DocumentChunk) {
			defer wg.Done()
			defer func() { <-sem }()
			for i, chunk := range batch {
				if ctx.Err() != nil {
					return
				}
				rg, err := s.extractor.Extract(ctx, chunk, entityTypes, relationTypes)
				if err != nil {
					logx.Warnf("抽取文档块图谱失败 chunkId=%d: %v", chunk.ID, err)
					continue
				}
				raws[offset+i] = rg
			}
		}(start, chunks[start:end])
	}
	wg.Wait()
	return utils.Filter(raws, func(rg *extract.RawGraph) bool { return rg != nil })
}

// projectTypedChunks M3 图卡片投影：图提交成功后把实体/关系/社区卡向量化写入向量库。
// 未装配向量索引器 / 投影关闭 / 图非空时跳过；写入失败仅告警（增强依赖）。
func (s *Builder) projectTypedChunks(ctx context.Context, snapshot *entity.GraphSnapshot) {
	if s.vectorIndexer == nil || !s.config.TypedChunkProjectionEnabled || snapshot == nil || len(snapshot.Nodes) == 0 {
		return
	}
	cards := GenerateTypedCards(snapshot, s.config.TypedCardTopK)
	if len(cards) == 0 {
		return
	}
	if err := s.vectorIndexer.BuildVectors(ctx, cards); err != nil {
		logx.Warnf("图卡片投影失败（增强依赖，不阻断）: documentId=%d, err=%v", snapshot.DocumentId, err)
		return
	}
	logx.Infof("图卡片投影完成: documentId=%d, cards=%d", snapshot.DocumentId, len(cards))
}

// enhanceCommunitySummaries M3 LLM 社区报告：仅对前 CommunityReportTopN 个重要社区生成 LLM 摘要
// （预算闸门，0 关闭）。任何单社区失败均告警并保留抽取式摘要，不阻断后续环节。
func (s *Builder) enhanceCommunitySummaries(ctx context.Context, nodes []*entity.GraphNode, communities []*entity.GraphCommunity) {
	if s.reporter == nil || len(communities) == 0 {
		return
	}
	communities = utils.Limit(communities, s.config.CommunityReportTopN)
	if len(communities) == 0 {
		return
	}
	byKey := make(map[string]*entity.GraphNode, len(nodes))
	for _, n := range nodes {
		byKey[n.EntityKey] = n
	}
	for _, c := range communities {
		members := utils.Filter(c.MemberKeys, func(key string) bool {
			return byKey[key] != nil
		})
		memberNodes := utils.Map(members, func(key string) *entity.GraphNode { return byKey[key] })
		summary, err := s.reporter.Report(ctx, c, memberNodes)
		if err != nil || summary == "" {
			logx.Warnf("LLM 社区报告失败，回退抽取式摘要: community_no=%d, err=%v", c.CommunityNo, err)
			continue
		}
		c.Summary = summary
	}
}

// enforceLimits 成本闸门：实体/关系超上限时按 PageRank 降序 + entity_key 升序全序截断，并同步过滤端点被裁剪的关系。
func enforceLimits(nodes []*entity.GraphNode, edges []*entity.GraphEdge, maxEntities, maxRelations int, documentId int64) ([]*entity.GraphNode, []*entity.GraphEdge) {
	if maxEntities > 0 && len(nodes) > maxEntities {
		ranked := make([]*entity.GraphNode, len(nodes))
		copy(ranked, nodes)
		slices.SortFunc(ranked, func(a, b *entity.GraphNode) int {
			if c := cmp.Compare(b.PageRank, a.PageRank); c != 0 {
				return c
			}
			return cmp.Compare(a.EntityKey, b.EntityKey)
		})
		kept := ranked[:maxEntities]
		keySet := make(map[string]struct{}, len(kept))
		for _, n := range kept {
			keySet[n.EntityKey] = struct{}{}
		}
		edges = filterEdgesByKeys(edges, keySet)
		nodes = kept
		slices.SortFunc(nodes, func(a, b *entity.GraphNode) int { return cmp.Compare(a.EntityKey, b.EntityKey) })
		logx.Warnf("实体成本闸门截断: documentId=%d, 保留 %d/%d", documentId, len(nodes), maxEntities)
	}
	if maxRelations > 0 && len(edges) > maxRelations {
		ranked := make([]*entity.GraphEdge, len(edges))
		copy(ranked, edges)
		slices.SortFunc(ranked, func(a, b *entity.GraphEdge) int {
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

// resolveGraphTypes 加载知识库自定义的图谱类型白名单（实体类型 + 关系类型）
func (s *Builder) resolveGraphTypes(ctx context.Context, knowledgeBaseId int64) (entityTypes, relationTypes []string) {
	if s.typesProvider == nil || knowledgeBaseId <= 0 {
		return nil, nil
	}
	types, rels, err := s.typesProvider.GraphTypes(ctx, knowledgeBaseId)
	if err != nil {
		logx.Warnf("加载知识库图谱类型白名单失败 kbId=%d: %v，回退全局默认", knowledgeBaseId, err)
		return nil, nil
	}
	return types, rels
}
