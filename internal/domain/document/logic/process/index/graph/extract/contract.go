package extract

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// RawEntity LLM 抽取出的原始实体
type RawEntity struct {
	Name          string `json:"name"`          // 实体名（原文最具代表性表述）
	Type          string `json:"type"`          // 实体类型（受控白名单）
	Description   string `json:"description"`   // 一句话事实描述
	SourceChunkId int64  `json:"sourceChunkId"` // 证据锚点 chunk id（grounding）
	Quote         string `json:"quote"`         // 原文引用片段（grounding 校验：必须是 chunk 文本子串）
}

// RawRelation LLM 抽取出的原始关系
type RawRelation struct {
	Source        string `json:"source"`        // 头实体名
	Target        string `json:"target"`        // 尾实体名
	RelationType  string `json:"relationType"`  // 受控关系类型（白名单，未命中落 RELATED_TO）
	RelationName  string `json:"relationName"`  // 中文可读谓词
	Description   string `json:"description"`   // 关系描述
	SourceChunkId int64  `json:"sourceChunkId"` // 证据锚点 chunk id（grounding）
	Quote         string `json:"quote"`         // 原文谓词引用（事实权威，必须是 chunk 文本子串）
}

// RawGraph 单块抽取结果
type RawGraph struct {
	Entities  []RawEntity   `json:"entities"`
	Relations []RawRelation `json:"relations"`
}

// Extractor 图谱抽取器
type Extractor interface {
	Extract(ctx context.Context, chunk *entity.DocumentChunk) (*RawGraph, error)
}

// Tokenizer 中文分词接口（复用 gse），为抽取提示提供候选词
type Tokenizer interface {
	SegmentWords(text string) []string
}
