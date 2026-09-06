package storage

import (
	"cmp"
	"context"
	"fmt"
	"slices"
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
	key         string
	name        string
	normName    string
	aliases     []string
	description string
	rankBoost   float64
	documentId  int64
	taskId      int64
	kbId        int64
}

// SearchBySeed 图谱检索：种子实体命中 → n-hop 变长路径扩展 → 证据回流 → 投影证据块。
func (s *Neo4jGraphStore) SearchBySeed(ctx context.Context, input *channel.Input) ([]*vo.DocumentChunk, error) {
	if s.driver == nil || len(input.DocumentIds) == 0 || strings.TrimSpace(input.Question) == "" {
		return nil, nil
	}
	hops := utils.Ternary(input.Hops <= 0, 2, input.Hops)
	// 1. 种子实体识别：将注入的种子实体并入匹配文本（LLM/Agent 实体优先命中），再叠加 gse 分词
	candidates, err := s.loadEntityCandidates(ctx, input.DocumentIds)
	if err != nil {
		return nil, err
	}
	matchText := input.Question
	if len(input.Entities) > 0 {
		matchText = strings.Join(append([]string{input.Question}, utils.FilterBlank(input.Entities)...), " ")
	}
	terms := segmentTerms(s.segmenter, matchText)
	seeds := matchSeedEntities(candidates, matchText, terms, input.TopK)
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
	slices.SortStableFunc(chunks, func(a, b *vo.DocumentChunk) int {
		return cmp.Compare(b.Score, a.Score)
	})
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
			rec := res.Record().AsMap()
			c := &entityCandidate{
				key:         utils.MapValue[string](rec, "key"),
				name:        utils.MapValue[string](rec, "name"),
				normName:    utils.MapValue[string](rec, "normalized"),
				aliases:     utils.MapValue[[]string](rec, "aliases"),
				description: utils.MapValue[string](rec, "description"),
				rankBoost:   utils.MapValue[float64](rec, "rank_boost"),
				documentId:  utils.MapValue[int64](rec, "document_id"),
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

// matchSeedEntities 打分选取种子实体：名称/别名/描述命中 + rankBoost 加权，取 max(topK*4, 12) 个作为种子。
func matchSeedEntities(candidates []*entityCandidate, question string, terms []string, topK int) []string {
	if len(candidates) == 0 {
		return nil
	}
	scores := make(map[*entityCandidate]float64, len(candidates))
	scored := make([]*entityCandidate, 0, len(candidates))
	for _, c := range candidates {
		if score := seedScore(c, question, terms); score > 0 {
			scores[c] = score
			scored = append(scored, c)
		}
	}
	if len(scored) == 0 {
		return nil
	}
	slices.SortFunc(scored, func(a, b *entityCandidate) int {
		if c := cmp.Compare(scores[b], scores[a]); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	})
	limit := topK * graphSeedTopKFactor
	if limit < graphMinSeedCount {
		limit = graphMinSeedCount
	}
	return utils.Map(utils.Limit(scored, limit), func(c *entityCandidate) string { return c.key })
}

// seedScore 实体种子打分（对齐 GraphRAG-简化落地方案 3.4 scoreEntity）：
// 名称命中 1.0 + 别名命中 0.7 + 描述命中 0.12×命中term数 + rankBoost×0.45，cap 2.0；
// 匹配为「双向子串 + 忽略大小写」，问题点名实体名与实体文本覆盖问题词均计命中。
// 构建侧 Name 取自 Aliases 的最长项（Name ∈ Aliases），名称命中必然伴生同义别名命中，
// 因此别名仅作为名称未命中时的独立后备信号，避免别名分对名称命中实体恒叠加而失去区分度。
func seedScore(c *entityCandidate, question string, terms []string) float64 {
	queries := append([]string{question}, terms...)
	score := 0.0
	if textHit(queries, c.name) || textHit(queries, c.normName) {
		score += 1.0
	} else {
		for _, alias := range c.aliases {
			if utils.EqualsIgnoreCase(alias, c.name) || utils.EqualsIgnoreCase(alias, c.normName) {
				continue
			}
			if textHit(queries, alias) {
				score += 0.7
				break
			}
		}
	}
	if hits := textHits(queries, c.description); hits > 0 {
		score += 0.12 * float64(hits)
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

// textHit 判定候选文本与任一查询词存在双向子串命中（忽略大小写）
func textHit(terms []string, candidate string) bool {
	return textHits(terms, candidate) > 0
}

// textHits 统计候选文本被命中的查询词个数（忽略大小写，候选含查询词或查询词含候选即命中）
func textHits(terms []string, candidate string) int {
	if strings.TrimSpace(candidate) == "" {
		return 0
	}
	seen := make(map[string]struct{}, len(terms))
	n := 0
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		if utils.ContainsFold(candidate, t) || utils.ContainsFold(t, candidate) {
			n++
		}
	}
	return n
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
			rec := res.Record().AsMap()
			relAny, ok := rec["rel"]
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
				sourceName:   utils.MapValue[string](rec, "source"),
				targetName:   utils.MapValue[string](rec, "target"),
				relationType: rel.Type,
				weight:       utils.MapValue[float64](rel.Props, "weight"),
				rankBoost:    utils.MapValue[float64](rel.Props, "rank_boost"),
				predicate:    utils.MapValue[string](rel.Props, "predicate_quote"),
				description:  utils.MapValue[string](rel.Props, "description"),
				chunkIds:     utils.MapValue[[]int64](rel.Props, "chunk_ids"),
				quoteTexts:   utils.MapValue[[]string](rel.Props, "quote_texts"),
				documentId:   utils.MapValue[int64](rec, "document_id"),
				taskId:       utils.MapValue[int64](rec, "task_id"),
				kbId:         utils.MapValue[int64](rec, "kb_id"),
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
			OriginalSnippet: r.firstQuote(),
			SourceType:      "GRAPH_RAG",
			Channel:         "graph_rag",
			TaskId:          r.taskId,
			DocumentId:      r.documentId,
			KnowledgeBaseId: r.kbId,
		})
	}
	slices.SortFunc(chunks, func(a, b *vo.DocumentChunk) int {
		return cmp.Compare(b.Score, a.Score)
	})
	if len(chunks) > topK {
		chunks = chunks[:topK]
	}
	return chunks
}

func (r *relationalEvidence) firstQuote() string {
	if r == nil {
		return ""
	}
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
			rec := res.Record().AsMap()
			title := utils.MapValue[string](rec, "title")
			if title == "" {
				continue
			}
			chunks := &vo.DocumentChunk{
				ID:              "graph-community-" + utils.MapValue[string](rec, "no"),
				Score:           utils.MapValue[float64](rec, "rank_boost") + 0.2,
				Content:         title + "\n" + utils.MapValue[string](rec, "summary"),
				OriginalSnippet: title,
				SourceType:      "GRAPH_RAG",
				Channel:         "graph_rag",
				TaskId:          utils.MapValue[int64](rec, "task_id"),
				DocumentId:      utils.MapValue[int64](rec, "document_id"),
				KnowledgeBaseId: utils.MapValue[int64](rec, "kb_id"),
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
       e.aliases AS aliases, e.description AS description,
       e.rank_boost AS rank_boost,
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
