// GraphRAG Neo4j 约束与索引脚本（对齐《GraphRAG-简化落地方案》§5.2）
// 由 svc 在启动时读取并幂等执行（全部 IF NOT EXISTS，重复执行安全）
// 注意 1：Neo4j 一次查询仅允许一条语句，故本脚本按分号拆分后逐条执行；
//         请以 ; 结束每条语句，注释行以 // 开头（执行前被剥离）

// 唯一性约束：全删全插幂等的基础，同时为 MERGE 提供 backing index
CREATE CONSTRAINT entity_unique IF NOT EXISTS FOR (e:Entity) REQUIRE (e.document_id, e.task_id, e.entity_key) IS UNIQUE;
CREATE CONSTRAINT community_unique IF NOT EXISTS FOR (c:Community) REQUIRE (c.document_id, c.task_id, c.community_no) IS UNIQUE;

// 检索路径索引
CREATE INDEX entity_name IF NOT EXISTS FOR (e:Entity) ON (e.normalized_name);
CREATE INDEX entity_doc IF NOT EXISTS FOR (e:Entity) ON (e.document_id, e.task_id);
CREATE INDEX entity_rank IF NOT EXISTS FOR (e:Entity) ON (e.rank_boost);
CREATE INDEX community_doc IF NOT EXISTS FOR (c:Community) ON (c.document_id, c.task_id);
