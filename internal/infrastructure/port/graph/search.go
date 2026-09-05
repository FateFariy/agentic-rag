package graph

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Segmenter 中文分词接口（种子实体识别，复用 gse）
type Segmenter interface {
	SegmentWords(text string) []string
}

const (
	graphSeedTopKFactor = 4
	graphMinSeedCount   = 12
)

// entityCandidate 文档内实体检索候选
type entityCandidate struct {
	key        string
	name       string
	normName   string
	aliases    []string
	rankBoost  float64
	documentId int64
	taskId     int64
	kbId       int64
}

// SearchBySeed 图谱检索：种子实体命中 → n-hop 变长路径扩展 → 证据回流 → 投影证据块。
func (s *Neo4jGraphStore) SearchBySeed(ctx context.Context, input *channel.Input) ([]*vo.DocumentChunk, error) {
	if s.driver == nil || len(input.DocumentIds) == 0 || strings.TrimSpace(input.Question) == "" {
		return nil, nil
	}
	hops := utils.Ternary(input.Hops <= 0, 2, input.Hops)
	// 1. 种子实体识别
	candidates, err := s.loadEntityCandidates(ctx, input.DocumentIds)
	if err != nil {
		return nil, err
	}
	terms := segmentTerms(s.segmenter, input.Question)
	seeds := matchSeedEntities(candidates, input.Question, terms, input.TopK)
	if len(seeds) == 0 {
		return nil, nil
	}
	// 2. n-hop 变长路径扩展（携带证据属性）
	relations, err := s.expandRelationalEvidence(ctx, input.DocumentIds, seeds, hops, input.TopK)
	if err != nil {
		return nil, err
	}
	chunks := projectEvidenceChunks(relations, input.TopK)
	// 3. 社区召回（主题级聚合，无 chunk 锚点 → 合成负 ID）
	if comms, cErr := s.searchCommunities(ctx, input.DocumentIds, terms, input.TopK); cErr == nil && len(comms) > 0 {
		chunks = append(chunks, comms...)
	}
	sort.SliceStable(chunks, func(i, j int) bool { return chunks[i].Score > chunks[j].Score })
	if len(chunks) > input.TopK {
		chunks = chunks[:input.TopK]
	}
	return chunks, nil
}

func segmentTerms(segmenter Segmenter, question string) []string {
	if segmenter == nil {
		return []string{question}
	}
	return segmenter.SegmentWords(question)
}

// loadEntityCandidates 加载指定文档全集内的实体候选（含别名与血缘）
func (s *Neo4jGraphStore) loadEntityCandidates(ctx context.Context, documentIds []int64) ([]*entityCandidate, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, e := tx.Run(ctx, entityCandidateCypher, map[string]any{"docIds": documentIds})
		if e != nil {
			return nil, e
		}
		var out []*entityCandidate
		for res.Next(ctx) {
			rec := res.Record()
			docId, _ := rec.Get("document_id")
			c := &entityCandidate{
				key:        recStr(rec, "key"),
				name:       recStr(rec, "name"),
				normName:   recStr(rec, "normalized"),
				aliases:    recStrSlice(rec, "aliases"),
				rankBoost:  recFloat(rec, "rank_boost"),
				documentId: utils.Convert[int64](docId),
			}
			if c.key == "" {
				continue
			}
			out = append(out, c)
		}
		return out, res.Err()
	})
	if err != nil {
		return nil, err
	}
	list, _ := result.([]*entityCandidate)
	return list, nil
}

// matchSeedEntities 打分选取种子实体：名称/别名命中（问题子串或分词命中）+ rankBoost 加权。
// 取 max(topK*4, 12) 个作为种子。
func matchSeedEntities(candidates []*entityCandidate, question string, terms []string, topK int) []string {
	if len(candidates) == 0 {
		return nil
	}
	scored := make([]*entityCandidate, 0, len(candidates))
	for _, c := range candidates {
		score := seedScore(c, question, terms)
		if score <= 0 {
			continue
		}
		c.rankBoost = score // 复用 rankBoost 槽位存种子分，避免新字段
		scored = append(scored, c)
	}
	if len(scored) == 0 {
		return nil
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].rankBoost != scored[j].rankBoost {
			return scored[i].rankBoost > scored[j].rankBoost
		}
		return scored[i].key < scored[j].key
	})
	limit := topK * graphSeedTopKFactor
	if limit < graphMinSeedCount {
		limit = graphMinSeedCount
	}
	if len(scored) > limit {
		scored = scored[:limit]
	}
	keys := make([]string, 0, len(scored))
	for _, c := range scored {
		keys = append(keys, c.key)
	}
	return keys
}

func seedScore(c *entityCandidate, question string, terms []string) float64 {
	score := 0.0
	if textHit(question, terms, c.name) || textHit(question, terms, c.normName) {
		score += 1.0
	}
	for _, alias := range c.aliases {
		if textHit(question, terms, alias) {
			score += 0.7
			break
		}
	}
	if score <= 0 {
		return 0
	}
	score += c.rankBoost * 0.45
	if score > 2.0 {
		score = 2.0
	}
	return score
}

// textHit 判定 candidate 文本是否被问题或其分词命中（子串匹配，candidate 非空）
func textHit(question string, terms []string, candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	if strings.Contains(question, candidate) {
		return true
	}
	for _, term := range terms {
		if term != "" && strings.Contains(candidate, term) {
			return true
		}
	}
	return false
}

// relationalEvidence 一条关系证据
type relationalEvidence struct {
	sourceKey    string
	sourceName   string
	targetKey    string
	targetName   string
	relationType string
	weight       float64
	rankBoost    float64
	predicate    string
	description  string
	chunkIds     []int64
	quoteTexts   []string
	documentId   int64
	taskId       int64
	kbId         int64
}

// expandRelationalEvidence 变长路径 1..hops 扩展，逐条返回关系证据（按路径对展开）
func (s *Neo4jGraphStore) expandRelationalEvidence(ctx context.Context, documentIds []int64, seedKeys []string, hops, topK int) ([]*relationalEvidence, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, e := tx.Run(ctx, relationPathCypher(hops), map[string]any{
			"docIds":   documentIds,
			"seedKeys": seedKeys,
			"maxHops":  hops,
			"limit":    topK * 40,
		})
		if e != nil {
			return nil, e
		}
		var out []*relationalEvidence
		seen := map[string]struct{}{}
		for res.Next(ctx) {
			rec := res.Record()
			relAny, ok := rec.Get("rel")
			if !ok {
				continue
			}
			rel, ok := relAny.(neo4j.Relationship)
			if !ok {
				continue
			}
			if _, dup := seen[rel.ElementId]; dup {
				continue
			}
			seen[rel.ElementId] = struct{}{}
			out = append(out, &relationalEvidence{
				sourceName:   recStr(rec, "source"),
				targetName:   recStr(rec, "target"),
				relationType: rel.Type,
				weight:       utils.MapValue[float64](rel.Props, "weight"),
				rankBoost:    utils.MapValue[float64](rel.Props, "rank_boost"),
				predicate:    utils.MapValue[string](rel.Props, "predicate_quote"),
				description:  utils.MapValue[string](rel.Props, "description"),
				chunkIds:     utils.MapValue[[]int64](rel.Props, "chunk_ids"),
				quoteTexts:   utils.MapValue[[]string](rel.Props, "quote_texts"),
				documentId:   recInt(rec, "document_id"),
				taskId:       recInt(rec, "task_id"),
				kbId:         recInt(rec, "kb_id"),
			})
		}
		return out, res.Err()
	})
	if err != nil {
		return nil, err
	}
	list, _ := result.([]*relationalEvidence)
	return list, nil
}

// projectEvidenceChunks 证据回流：每条关系投影为一个证据块；
// 有 chunk 锚点的用真实 chunk id（引用可定位），无锚点用合成负 ID（跳过父块提升）。
func projectEvidenceChunks(relations []*relationalEvidence, topK int) []*vo.DocumentChunk {
	if len(relations) == 0 {
		return nil
	}
	chunks := make([]*vo.DocumentChunk, 0, len(relations))
	negID := -1
	for _, r := range relations {
		content := r.predicate
		if content == "" {
			content = r.description
		}
		if len(r.quoteTexts) > 0 {
			content = strings.Join(r.quoteTexts, "\n")
		}
		if content == "" {
			continue
		}
		var id string
		if len(r.chunkIds) > 0 {
			id = strconv.FormatInt(r.chunkIds[0], 10)
		} else {
			id = strconv.FormatInt(int64(negID), 10)
			negID--
		}
		score := r.rankBoost + r.weight*0.55
		chunks = append(chunks, &vo.DocumentChunk{
			ID:              id,
			Score:           score,
			Content:         content,
			OriginalSnippet: firstQuote(r),
			SourceType:      "GRAPH_RAG",
			Channel:         "graph_rag",
			TaskId:          r.taskId,
			DocumentId:      r.documentId,
			KnowledgeBaseId: r.kbId,
		})
	}
	sort.Slice(chunks, func(i, j int) bool { return chunks[i].Score > chunks[j].Score })
	if len(chunks) > topK {
		chunks = chunks[:topK]
	}
	return chunks
}

func firstQuote(r *relationalEvidence) string {
	if len(r.quoteTexts) > 0 {
		return r.quoteTexts[0]
	}
	if r.predicate != "" {
		return r.predicate
	}
	return r.description
}

// searchCommunities 社区召回：title/summary 命中分词，合成主题级证据块（无 chunk 锚点）
func (s *Neo4jGraphStore) searchCommunities(ctx context.Context, documentIds []int64, terms []string, topK int) ([]*vo.DocumentChunk, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, e := tx.Run(ctx, communityMatchCypher, map[string]any{
			"docIds": documentIds,
			"terms":  terms,
			"limit":  topK,
		})
		if e != nil {
			return nil, e
		}
		var out []*vo.DocumentChunk
		for res.Next(ctx) {
			rec := res.Record()
			title := recStr(rec, "title")
			if title == "" {
				continue
			}
			chunks := &vo.DocumentChunk{
				ID:              "graph-community-" + recStr(rec, "no"),
				Score:           recFloat(rec, "rank_boost") + 0.2,
				Content:         title + "\n" + recStr(rec, "summary"),
				OriginalSnippet: title,
				SourceType:      "GRAPH_RAG",
				Channel:         "graph_rag",
				TaskId:          recInt(rec, "task_id"),
				DocumentId:      recInt(rec, "document_id"),
				KnowledgeBaseId: recInt(rec, "kb_id"),
			}
			out = append(out, chunks)
		}
		return out, res.Err()
	})
	if err != nil {
		return nil, err
	}
	list, _ := result.([]*vo.DocumentChunk)
	return list, nil
}

// ===================== Cypher =====================

const entityCandidateCypher = `MATCH (e:Entity)
WHERE e.document_id IN $docIds
RETURN e.entity_key AS key, e.name AS name, e.normalized_name AS normalized,
       e.aliases AS aliases, e.rank_boost AS rank_boost,
       e.document_id AS document_id, e.task_id AS task_id, e.kb_id AS kb_id
LIMIT 2000`

func relationPathCypher(hops int) string {
	return fmt.Sprintf(`MATCH (seed:Entity)
WHERE seed.document_id IN $docIds AND seed.entity_key IN $seedKeys
MATCH path = (seed)-[rels*1..%d]-(linked:Entity)
WHERE linked.document_id IN $docIds
WITH nodes(path) AS ns, rels(path) AS rs
UNWIND range(1, size(ns)-1) AS i
WITH ns[i-1] AS a, rs[i-1] AS rel, ns[i] AS b
RETURN rel AS rel, a.name AS source, b.name AS target,
       a.document_id AS document_id, a.task_id AS task_id, a.kb_id AS kb_id
LIMIT $limit`, hops)
}

const communityMatchCypher = `MATCH (c:Community)-[:IN_COMMUNITY]-(e:Entity)
WHERE c.document_id IN $docIds
  AND any(t IN $terms WHERE c.title CONTAINS t OR c.summary CONTAINS t)
RETURN c.community_no AS no, c.title AS title, c.summary AS summary,
       c.rank_boost AS rank_boost, c.document_id AS document_id,
       c.task_id AS task_id, c.kb_id AS kb_id
LIMIT $limit`

// ===================== record 取值 helper =====================

func recStr(rec *neo4j.Record, key string) string {
	if rec == nil {
		return ""
	}
	v, ok := rec.Get(key)
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func recInt(rec *neo4j.Record, key string) int64 {
	if rec == nil {
		return 0
	}
	v, ok := rec.Get(key)
	if !ok {
		return 0
	}
	return utils.Convert[int64](v)
}

func recFloat(rec *neo4j.Record, key string) float64 {
	if rec == nil {
		return 0
	}
	v, ok := rec.Get(key)
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

func recStrSlice(rec *neo4j.Record, key string) []string {
	if rec == nil {
		return nil
	}
	v, ok := rec.Get(key)
	if !ok || v == nil {
		return nil
	}
	switch list := v.(type) {
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return list
	}
	return nil
}
