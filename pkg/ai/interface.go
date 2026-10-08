package ai

import (
	"context"

	"github.com/swiftbit/know-agent/pkg/common"
)

// ChatModel 大模型生成端口
type ChatModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...common.Option) (string, error)

	GenerateWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (string, error)

	Stream(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (<-chan string, error)
}

// PromptRenderer 负责将 sourceText 渲染为大模型提示词
type PromptRenderer interface {
	// Render 渲染提示词
	Render(templateName string, variables map[string]any) (string, error)
}
