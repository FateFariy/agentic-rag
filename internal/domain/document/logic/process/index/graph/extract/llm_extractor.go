package extract

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// LLMExtractor 基于大模型的实体/关系抽取器
type LLMExtractor struct {
	chatModel model.ChatModel
	renderer  adapter.PromptRenderer
	tokenizer Tokenizer
}

// NewLLMExtractor 创建大模型抽取器
func NewLLMExtractor(chatModel model.ChatModel, renderer adapter.PromptRenderer, tokenizer Tokenizer) *LLMExtractor {
	return &LLMExtractor{
		chatModel: chatModel,
		renderer:  renderer,
		tokenizer: tokenizer,
	}
}

// Extract 抽取单个块
func (e *LLMExtractor) Extract(ctx context.Context, chunk *entity.DocumentChunk) (*RawGraph, error) {
	text := strings.TrimSpace(chunk.ChunkText)
	if text == "" {
		return &RawGraph{}, nil
	}
	userPrompt, err := e.renderUser(text)
	if err != nil {
		return nil, err
	}
	resp, err := e.chatModel.Generate(ctx, "", userPrompt)
	if err != nil {
		return nil, err
	}
	return parseRawGraph(resp, chunk.ID)
}

func (e *LLMExtractor) renderUser(text string) (string, error) {
	hint := ""
	if e.tokenizer != nil {
		if tokens := e.tokenizer.SegmentWords(text); len(tokens) > 0 {
			hint = strings.Join(utils.Limit(tokens, 80), " ")
		}
	}
	return e.renderer.Render(enum.GraphExtract, map[string]any{
		"text":      text,
		"tokenHint": hint,
		"types":     sortedKeys(enum.GraphRagEntityTypes),
		"relations": sortedKeys(enum.GraphRagRelationWhitelist),
	})
}

func parseRawGraph(resp string, chunkId int64) (*RawGraph, error) {
	raw := &RawGraph{}
	cleaned := extractJSONBlock(resp)
	if cleaned == "" {
		return raw, nil
	}
	if err := json.Unmarshal([]byte(cleaned), raw); err != nil {
		return raw, err
	}
	for i := range raw.Entities {
		raw.Entities[i].SourceChunkId = chunkId
	}
	for i := range raw.Relations {
		raw.Relations[i].SourceChunkId = chunkId
	}
	return raw, nil
}

func extractJSONBlock(s string) string {
	s = strings.TrimSpace(s)
	// 去除 ```json ... ``` 围栏
	if start := strings.Index(s, "```"); start >= 0 {
		s = s[start+3:]
		if end := strings.Index(s, "```"); end >= 0 {
			s = s[:end]
		}
		s = strings.TrimSpace(s)
		if strings.HasPrefix(s, "json") {
			s = strings.TrimSpace(s[4:])
		}
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
