package extract

import (
	"context"
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// LLMExtractor 基于大模型的实体/关系抽取器
type LLMExtractor struct {
	chatModel     model.ChatModel
	renderer      adapter.PromptRenderer
	tokenizer     Tokenizer
	chunkMaxChars int // 单 chunk 抽取截断上限
}

// NewLLMExtractor 创建大模型抽取器
func NewLLMExtractor(chatModel model.ChatModel, renderer adapter.PromptRenderer, tokenizer Tokenizer, chunkMaxChars int) *LLMExtractor {
	return &LLMExtractor{
		chatModel:     chatModel,
		renderer:      renderer,
		tokenizer:     tokenizer,
		chunkMaxChars: chunkMaxChars,
	}
}

// Extract 抽取单个块（超长块按 ChunkMaxChars 截断，控制 LLM 输入成本）
func (e *LLMExtractor) Extract(ctx context.Context, chunk *entity.DocumentChunk, entityTypes, relationTypes []string) (*RawGraph, error) {
	text := strings.TrimSpace(chunk.ChunkText)
	if text == "" {
		return &RawGraph{}, nil
	}
	if e.chunkMaxChars > 0 {
		text = utils.Truncate(text, e.chunkMaxChars)
	}
	if len(entityTypes) == 0 {
		entityTypes = utils.MapKeys(enum.GraphRagEntityTypes)
	}
	if len(relationTypes) == 0 {
		relationTypes = utils.MapKeys(enum.GraphRagRelationWhitelist)
	}

	raw, err := e.generateAndParse(ctx, text, chunk.ID, entityTypes, relationTypes)
	if err != nil {
		return nil, err
	}

	// 契约 + grounding 校验：quote 非原文子串/缺失的条目直接丢弃（不整体失败）
	ValidateGrounding(raw, text)
	return raw, nil
}

// generateAndParse 渲染提示词 → 调用模型 → 解析响应
func (e *LLMExtractor) generateAndParse(ctx context.Context, text string, chunkId int64, types, relations []string) (*RawGraph, error) {
	hint := ""
	if e.tokenizer != nil {
		if tokens := e.tokenizer.SegmentWords(text); len(tokens) > 0 {
			hint = strings.Join(utils.Limit(tokens, 80), " ")
		}
	}
	userPrompt, err := e.renderer.Render(enum.GraphExtract, map[string]any{
		"text":      text,
		"tokenHint": hint,
		"types":     types,
		"relations": relations,
	})
	if err != nil {
		return nil, fmt.Errorf("render prompt: %w", err)
	}

	resp, err := e.chatModel.Generate(ctx, "", userPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm generate: %w", err)
	}

	raw := &RawGraph{}
	if err = utils.Unmarshal(resp, raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	for i := range raw.Entities {
		raw.Entities[i].SourceChunkId = chunkId
	}
	for i := range raw.Relations {
		raw.Relations[i].SourceChunkId = chunkId
	}

	return raw, nil
}
