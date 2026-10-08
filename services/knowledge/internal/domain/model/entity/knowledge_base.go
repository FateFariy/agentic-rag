package entity

import (
	"encoding/json"
)

type KnowledgeBase struct {
	ID                       int64           `gorm:"column:id;"`                                             // 知识库ID
	BaseName                 string          `gorm:"column:base_name;"`                                      // 知识库名称
	Description              string          `gorm:"column:description;"`                                    // 描述
	EmbeddingModel           string          `gorm:"column:embedding_model;"`                                // 嵌入模型
	RetrievalConfigJson      json.RawMessage `gorm:"column:retrieval_config_json;type:json;serializer:json"` // 检索配置JSON
	GraphRagConfigJson       json.RawMessage `gorm:"column:graph_rag_config_json;type:json;serializer:json"` // 图谱RAG配置JSON
	MetadataFilterJson       json.RawMessage `gorm:"column:metadata_filter_json;type:json;serializer:json"`  // 元数据过滤JSON
	EntityTypes              []string        `gorm:"column:entity_types;type:json;serializer:json"`          // 图谱实体类型白名单
	RelationTypes            []string        `gorm:"column:relation_types;type:json;serializer:json"`        // 图谱关系类型白名单
	IsDefault                *int            `gorm:"column:is_default;type:tinyint"`                         // 是否默认
	SortOrder                int             `gorm:"column:sort_order;type:int"`                             // 排序
	DocumentCount            int             `gorm:"column:document_count;type:int"`                         // 文档数量
	RetrievableDocumentCount int             `gorm:"column:retrievable_document_count;type:int"`             // 可检索文档数量
}
