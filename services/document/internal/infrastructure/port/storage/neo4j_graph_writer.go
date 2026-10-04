package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	neo4jsdk "github.com/swiftbit/know-agent/pkg/neo4j"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
)

// DocumentGraphStore 图谱存储
type DocumentGraphStore struct {
	*neo4jsdk.Neo4jGraphStore
}

// NewDocumentGraphStore 包装根 Neo4jGraphStore 形成 document 图谱存储
func NewDocumentGraphStore(inner *neo4jsdk.Neo4jGraphStore) *DocumentGraphStore {
	return &DocumentGraphStore{Neo4jGraphStore: inner}
}

// ReplaceDocumentGraph 事务化全删全插（按 document_id + task_id 幂等）：
// DETACH DELETE 旧图 → MERGE Entity/Community 节点 → 动态受控关系 → :IN_COMMUNITY 边
func (s *DocumentGraphStore) ReplaceDocumentGraph(ctx context.Context, snapshot *entity.GraphSnapshot) error {
	return s.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) error {
		params := map[string]any{
			"documentId": snapshot.DocumentId,
			"taskId":     snapshot.TaskId,
			"kbId":       snapshot.KnowledgeBaseId,
		}
		// 1. 全删全插：清空该文档旧图（DETACH 同时删除关系与社区归属边）
		if _, e := tx.Run(ctx, deleteDocumentGraphCypher, params); e != nil {
			return e
		}
		// 2. 实体节点（node key 幂等 MERGE）
		if len(snapshot.Nodes) > 0 {
			entityParams := cloneParams(params)
			entityParams["entities"] = toEntityParams(snapshot.Nodes)
			if _, e := tx.Run(ctx, mergeEntitiesCypher, entityParams); e != nil {
				return e
			}
		}
		// 3. 社区节点
		if len(snapshot.Communities) > 0 {
			commParams := cloneParams(params)
			commParams["communities"] = toCommunityParams(snapshot.Communities)
			if _, e := tx.Run(ctx, mergeCommunitiesCypher, commParams); e != nil {
				return e
			}
		}
		// 4. 语义关系：所有关系类型合并为单条 Cypher 批量落库（每种类型一个静态 CALL 块，零插件、无注入面）
		relGroups := groupRelationsByType(snapshot.Edges)
		if len(relGroups) > 0 {
			relParams := cloneParams(params)
			var relBlocks strings.Builder
			for i, group := range relGroups {
				if i > 0 {
					relBlocks.WriteString("\n")
				}
				relBlocks.WriteString(fmt.Sprintf(mergeRelsCypherTpl, group.relationType, group.relationType))
			}
			relParams["rels"] = toRelParams(snapshot.Edges)
			if _, e := tx.Run(ctx, "UNWIND $rels AS r\n"+relBlocks.String(), relParams); e != nil {
				return e
			}
		}
		// 5. 社区归属 :IN_COMMUNITY 边
		if len(snapshot.Communities) > 0 {
			memberParams := cloneParams(params)
			memberParams["members"] = flattenCommunityMembers(snapshot.Communities)
			if _, e := tx.Run(ctx, linkMembersCypher, memberParams); e != nil {
				return e
			}
		}
		return nil
	})
}

// DeleteDocumentGraph 删除文档全量图谱（含全部相关节点与关系，幂等）
func (s *DocumentGraphStore) DeleteDocumentGraph(ctx context.Context, documentId int64) error {
	return s.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) error {
		_, e := tx.Run(ctx, "MATCH (n {document_id: $documentId}) DETACH DELETE n",
			map[string]any{"documentId": documentId})
		return e
	})
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

// 动态受控关系类型静态 CALL 子查询块（relationType 来自白名单，拼接安全）
// 每个块由 WHERE r.relation_type 守卫，多个块在单条 Cypher 内批量落库所有关系类型
const mergeRelsCypherTpl = `CALL {
  WITH r
  MATCH (a:Entity {document_id: $documentId, task_id: $taskId, entity_key: r.source_key})
  MATCH (b:Entity {document_id: $documentId, task_id: $taskId, entity_key: r.target_key})
  WHERE r.relation_type = '%s'
  MERGE (a)-[rel:%s]->(b)
  SET rel.weight = r.weight,
      rel.rank_boost = r.rank_boost,
      rel.predicate_quote = r.predicate_quote,
      rel.description = r.description,
      rel.chunk_ids = r.chunk_ids,
      rel.quote_texts = r.quote_texts,
      rel.kb_id = $kbId
}`
