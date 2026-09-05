package extract

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// RawEntity LLM 抽取出的原始实体
type RawEntity struct {
	Name          string `json:"name"`          // 实体名（原文最具代表性表述）
	Type          string `json:"type"`          // 实体类型（受控白名单）
	Description   string `json:"description"`   // 一句话事实描述
	SourceChunkId int64  `json:"sourceChunkId"` // 证据锚点 chunk id（grounding）
	Quote         string `json:"quote"`         // 原文引用片段（grounding：必须为 chunk 文本子串，否则丢弃）
}

// RawRelation LLM 抽取出的原始关系
type RawRelation struct {
	Source        string `json:"source"`        // 头实体名
	Target        string `json:"target"`        // 尾实体名
	RelationType  string `json:"relationType"`  // 受控关系类型（白名单，未命中落 RELATED_TO）
	RelationName  string `json:"relationName"`  // 中文可读谓词
	Description   string `json:"description"`   // 关系描述
	SourceChunkId int64  `json:"sourceChunkId"` // 证据锚点 chunk id（grounding）
	Quote         string `json:"quote"`         // 原文谓词引用（事实权威，必须为 chunk 文本子串，否则丢弃）
}

// RawGraph 单块抽取结果
type RawGraph struct {
	Entities         []RawEntity   `json:"entities"`
	Relations        []RawRelation `json:"relations"`
	DroppedEntities  int           // 被 grounding 校验丢弃的实体条数
	DroppedRelations int           // 被 grounding 校验丢弃的关系条数
}

// Extractor 图谱抽取器
type Extractor interface {
	Extract(ctx context.Context, chunk *entity.DocumentChunk) (*RawGraph, error)
}

// Tokenizer 中文分词接口（复用 gse），为抽取提示提供候选词
type Tokenizer interface {
	SegmentWords(text string) []string
}

// foldSpace 归一化空白（连续空白折叠为单空格），用于 quote 与原文的宽松子串匹配
func foldSpace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

// stripQuoteParens 去除 quote 两端常见包裹引号/书名号
func stripQuoteParens(s string) string {
	s = strings.TrimSpace(s)
	for _, p := range []struct {
		left, right string
	}{
		{`"`, `"`}, {`'`, `'`}, {`“`, `”`}, {`「`, `」`}, {`『`, `』`},
	} {
		if strings.HasPrefix(s, p.left) && strings.HasSuffix(s, p.right) && len([]rune(s)) > len([]rune(p.left)) {
			s = strings.TrimSpace(s[len(p.left) : len(s)-len(p.right)])
		}
	}
	return s
}

// isQuoteInText grounding 校验：quote 归一化后必须是 chunk 文本的（空白折叠）子串
func isQuoteInText(quote, chunkText string) bool {
	q := foldSpace(stripQuoteParens(quote))
	if q == "" {
		return false
	}
	return strings.Contains(foldSpace(chunkText), q)
}

// ValidateGrounding 契约校验 + grounding 校验：
// 必填缺失（实体名/关系端点）或 quote 缺失/非原文子串 → 丢弃该条，不影响其余条目。
// 返回丢弃计数写入 raw，供上层可观测降级统计。
func ValidateGrounding(raw *RawGraph, chunkText string) {
	if raw == nil {
		return
	}
	keptEntities := raw.Entities[:0]
	for _, e := range raw.Entities {
		if strings.TrimSpace(e.Name) == "" || !isQuoteInText(e.Quote, chunkText) {
			raw.DroppedEntities++
			continue
		}
		e.Name = strings.TrimSpace(e.Name)
		e.Quote = stripQuoteParens(e.Quote)
		keptEntities = append(keptEntities, e)
	}
	raw.Entities = keptEntities

	keptRelations := raw.Relations[:0]
	for _, r := range raw.Relations {
		if strings.TrimSpace(r.Source) == "" || strings.TrimSpace(r.Target) == "" || !isQuoteInText(r.Quote, chunkText) {
			raw.DroppedRelations++
			continue
		}
		r.Source = strings.TrimSpace(r.Source)
		r.Target = strings.TrimSpace(r.Target)
		r.Quote = stripQuoteParens(r.Quote)
		keptRelations = append(keptRelations, r)
	}
	raw.Relations = keptRelations
}
