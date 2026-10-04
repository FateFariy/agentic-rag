package model

import (
	"context"

	"github.com/swiftbit/know-agent/common"
)

// ChatModel 大模型生成端口
type ChatModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...common.Option) (string, error)
}
