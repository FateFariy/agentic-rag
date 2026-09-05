package storage

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/svc"
)

// Neo4jGraphStore 基于 Neo4j 的图谱存储与检索网关
type Neo4jGraphStore struct {
	driver   neo4j.DriverWithContext
	graphCfg *config.GraphRagConf
	db       string
}

// NewNeo4jGraphStore 创建 Neo4j 图谱存储
func NewNeo4jGraphStore(svcCtx *svc.ServiceContext, graphCfg *config.GraphRagConf) *Neo4jGraphStore {
	return &Neo4jGraphStore{
		driver:   svcCtx.Neo4j,
		graphCfg: graphCfg,
		db:       svcCtx.Config.Neo4j.Database,
	}
}

// ApplyConstraints 执行约束/索引脚本
func (s *Neo4jGraphStore) ApplyConstraints(ctx context.Context, cypher string) error {
	if s.driver == nil || cypher == "" {
		return nil
	}
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	_, err := session.Run(ctx, cypher, nil)
	return err
}

// CommitGraph 事务化全删全插（按 document_id + task_id 幂等）：
// DETACH DELETE 旧图 → MERGE Entity/Community 节点 → 动态受控关系 → :IN_COMMUNITY 边。
func (s *Neo4jGraphStore) CommitGraph(ctx context.Context, snapshot *entity.GraphSnapshot) error {
	if s.driver == nil {
		return fmt.Errorf("neo4j driver is nil")
	}
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		params := map[string]any{
			"documentId": snapshot.DocumentId,
			"taskId":     snapshot.TaskId,
			"kbId":       snapshot.KnowledgeBaseId,
		}
		// 1. 全删全插：清空该文档旧图（DETACH 同时删除关系与社区归属边）
		if _, e := tx.Run(ctx, deleteDocumentGraphCypher, params); e != nil {
			return nil, e
		}
		// 2. 实体节点（node key 幂等 MERGE）
		if len(snapshot.Nodes) > 0 {
			entityParams := cloneParams(params)
			entityParams["entities"] = toEntityParams(snapshot.Nodes)
			if _, e := tx.Run(ctx, mergeEntitiesCypher, entityParams); e != nil {
				return nil, e
			}
		}
		// 3. 社区节点
		if len(snapshot.Communities) > 0 {
			commParams := cloneParams(params)
			commParams["communities"] = toCommunityParams(snapshot.Communities)
			if _, e := tx.Run(ctx, mergeCommunitiesCypher, commParams); e != nil {
				return nil, e
			}
		}
		// 4. 语义关系：Go 侧按受控类型分组，每种类型一条静态 Cypher（零插件、无注入面）
		for _, group := range groupRelationsByType(snapshot.Edges) {
			relParams := cloneParams(params)
			relParams["rels"] = toRelParams(group.edges)
			if _, e := tx.Run(ctx, mergeRelsCypher(group.relationType), relParams); e != nil {
				return nil, e
			}
		}
		// 5. 社区归属 :IN_COMMUNITY 边
		if len(snapshot.Communities) > 0 {
			memberParams := cloneParams(params)
			memberParams["members"] = flattenCommunityMembers(snapshot.Communities)
			if _, e := tx.Run(ctx, linkMembersCypher, memberParams); e != nil {
				return nil, e
			}
		}
		return nil, nil
	})
	return err
}

// QueryGraph 查询 N 跳子图上下文（kb 维度种子命中 + 变长路径扩展）
func (s *Neo4jGraphStore) QueryGraph(ctx context.Context, req *vo.GraphQuery) (*vo.GraphContext, error) {
	if s.driver == nil {
		return nil, nil
	}
	if req == nil || len(req.Entities) == 0 {
		return nil, nil
	}
	hops := req.MaxHops
	if hops <= 0 {
		hops = s.graphCfg.MaxHops
	}
	if hops <= 0 {
		hops = 2
	}
	limit := req.Limit
	if limit <= 0 {
		limit = s.graphCfg.TopK
	}
	if limit <= 0 {
		limit = 30
	}
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, e := tx.Run(ctx, queryGraphCypher(hops), map[string]any{
			"kbIds": req.KnowledgeBaseIds,
			"names": req.Entities,
		})
		if e != nil {
			return nil, e
		}
		return collectGraphContext(ctx, res, limit)
	})
	if err != nil {
		return nil, err
	}
	gc, _ := result.(*vo.GraphContext)
	return gc, nil
}

// ===================== Cypher 语句 =====================

const deleteDocumentGraphCypher = `MATCH (n {document_id: $documentId, task_id: $taskId}) DETACH DELETE n`

const mergeEntitiesCypher = `UNWIND $entities AS e
MERGE (n:Entity {document_id: $documentId, task_id: $taskId, entity_key: e.entity_key})
SET n.name = e.name,
    n.normalized_name = e.normalized_name,
    n.entity_type = e.entity_type,
    n.description = e.description,
    n.aliases = e.aliases,
    n.mention_count = e.mention_count,
    n.pagerank = e.pagerank,
    n.rank_boost = e.rank_boost,
    n.degree = e.degree,
    n.kb_id = $kbId`

const mergeCommunitiesCypher = `UNWIND $communities AS c
MERGE (cm:Community {document_id: $documentId, task_id: $taskId, community_no: c.community_no})
SET cm.level = c.level,
    cm.title = c.title,
    cm.summary = c.summary,
    cm.rank_boost = c.rank_boost,
    cm.modularity = c.modularity,
    cm.member_count = c.member_count,
    cm.kb_id = $kbId`

const linkMembersCypher = `UNWIND $members AS m
MATCH (e:Entity {document_id: $documentId, task_id: $taskId, entity_key: m.entity_key})
MATCH (c:Community {document_id: $documentId, task_id: $taskId, community_no: m.community_no})
MERGE (e)-[:IN_COMMUNITY]->(c)`

// mergeRelsCypher 动态受控关系类型静态 Cypher（relationType 来自白名单，拼接安全）
func mergeRelsCypher(relationType string) string {
	return fmt.Sprintf(`UNWIND $rels AS r
MATCH (a:Entity {document_id: $documentId, task_id: $taskId, entity_key: r.source_key})
MATCH (b:Entity {document_id: $documentId, task_id: $taskId, entity_key: r.target_key})
MERGE (a)-[rel:%s]->(b)
SET rel.weight = r.weight,
    rel.rank_boost = r.rank_boost,
    rel.predicate_quote = r.predicate_quote,
    rel.description = r.description,
    rel.chunk_ids = r.chunk_ids,
    rel.quote_texts = r.quote_texts,
    rel.kb_id = $kbId`, relationType)
}

func queryGraphCypher(hops int) string {
	return fmt.Sprintf(`MATCH (start:Entity)
WHERE start.kb_id IN $kbIds
  AND (start.normalized_name IN $names OR start.name IN $names OR any(a IN start.aliases WHERE a IN $names))
MATCH path = (start)-[rels*1..%d]-(end:Entity)
WHERE end.kb_id IN $kbIds
RETURN nodes(path) AS ns, rels AS rs
LIMIT 200`, hops)
}

// ===================== 参数映射 =====================

func cloneParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func toEntityParams(nodes []*entity.GraphNode) []map[string]any {
	params := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		aliases := n.Aliases
		if aliases == nil {
			aliases = []string{}
		}
		params = append(params, map[string]any{
			"entity_key":      n.EntityKey,
			"name":            n.Name,
			"normalized_name": n.NormalizedName,
			"entity_type":     n.EntityType,
			"description":     n.Description,
			"aliases":         aliases,
			"mention_count":   n.MentionCount,
			"pagerank":        n.PageRank,
			"rank_boost":      n.RankBoost,
			"degree":          n.Degree,
		})
	}
	return params
}

func toCommunityParams(communities []*entity.GraphCommunity) []map[string]any {
	params := make([]map[string]any, 0, len(communities))
	for _, c := range communities {
		params = append(params, map[string]any{
			"community_no": c.CommunityNo,
			"level":        c.Level,
			"title":        c.Title,
			"summary":      c.Summary,
			"rank_boost":   c.RankBoost,
			"modularity":   c.Modularity,
			"member_count": c.MemberCount,
		})
	}
	return params
}

func toRelParams(edges []*entity.GraphEdge) []map[string]any {
	params := make([]map[string]any, 0, len(edges))
	for _, e := range edges {
		chunkIDs := e.ChunkIds
		if chunkIDs == nil {
			chunkIDs = []int64{}
		}
		quoteTexts := e.QuoteTexts
		if quoteTexts == nil {
			quoteTexts = []string{}
		}
		params = append(params, map[string]any{
			"source_key":      e.SourceKey,
			"target_key":      e.TargetKey,
			"weight":          e.Weight,
			"rank_boost":      e.RankBoost,
			"predicate_quote": e.PredicateQuote,
			"description":     e.Description,
			"chunk_ids":       chunkIDs,
			"quote_texts":     quoteTexts,
		})
	}
	return params
}

// relationGroup 按受控关系类型分组的关系集合
type relationGroup struct {
	relationType string
	edges        []*entity.GraphEdge
}

// groupRelationsByType 将关系按 RelationType 分组，保证每种类型用一条静态 Cypher 落库
func groupRelationsByType(edges []*entity.GraphEdge) []relationGroup {
	groups := make(map[string][]*entity.GraphEdge)
	for _, e := range edges {
		t := e.RelationType
		if t == "" {
			t = "RELATED_TO"
		}
		groups[t] = append(groups[t], e)
	}
	out := make([]relationGroup, 0, len(groups))
	for t, es := range groups {
		out = append(out, relationGroup{relationType: t, edges: es})
	}
	return out
}

func flattenCommunityMembers(communities []*entity.GraphCommunity) []map[string]any {
	var members []map[string]any
	for _, c := range communities {
		for _, key := range c.MemberKeys {
			members = append(members, map[string]any{
				"community_no": c.CommunityNo,
				"entity_key":   key,
			})
		}
	}
	return members
}

func collectGraphContext(ctx context.Context, res neo4j.ResultWithContext, limit int) (*vo.GraphContext, error) {
	gc := &vo.GraphContext{Source: "neo4j"}
	idName := map[int64]string{}
	nameSet := map[string]struct{}{}
	relSet := map[string]struct{}{}
	for res.Next(ctx) {
		rec := res.Record()
		if rec == nil {
			continue
		}
		if ns, ok := rec.Get("ns"); ok {
			if list, ok := ns.([]any); ok {
				for _, v := range list {
					n, ok := v.(neo4j.Node)
					if !ok {
						continue
					}
					name := propStr(n.Props, "name")
					if name == "" {
						continue
					}
					idName[n.Id] = name
					if _, seen := nameSet[name]; seen {
						continue
					}
					nameSet[name] = struct{}{}
					gc.Entities = append(gc.Entities, &vo.GraphContextEntity{
						Name:        name,
						Type:        propStr(n.Props, "entity_type"),
						Description: propStr(n.Props, "description"),
						Rank:        propFloat(n.Props, "pagerank"),
					})
				}
			}
		}
		if rs, ok := rec.Get("rs"); ok {
			if list, ok := rs.([]any); ok {
				for _, v := range list {
					r, ok := v.(neo4j.Relationship)
					if !ok {
						continue
					}
					src := idName[r.StartId]
					tgt := idName[r.EndId]
					if src == "" || tgt == "" {
						continue
					}
					key := src + "|" + r.Type + "|" + tgt
					if _, seen := relSet[key]; seen {
						continue
					}
					relSet[key] = struct{}{}
					gc.Relations = append(gc.Relations, &vo.GraphContextRelation{
						Source:      src,
						Target:      tgt,
						Relation:    r.Type,
						Description: propStr(r.Props, "description"),
						Weight:      propFloat(r.Props, "weight"),
					})
				}
			}
		}
	}
	if err := res.Err(); err != nil {
		return gc, err
	}
	if limit > 0 {
		if len(gc.Entities) > limit {
			gc.Entities = gc.Entities[:limit]
		}
		if len(gc.Relations) > limit {
			gc.Relations = gc.Relations[:limit]
		}
	}
	return gc, nil
}

func propStr(props map[string]any, key string) string {
	if v, ok := props[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func propFloat(props map[string]any, key string) float64 {
	if v, ok := props[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return n
		case int64:
			return float64(n)
		case int:
			return float64(n)
		}
	}
	return 0
}
