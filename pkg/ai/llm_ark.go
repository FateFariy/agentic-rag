package ai

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	einoschema "github.com/cloudwego/eino/schema"

	"github.com/swiftbit/know-agent/pkg/utils"
)

// NewArkChatModel 基于火山方舟（Ark）配置创建对话模型；失败直接 panic（启动期致命错误）
func NewArkChatModel(ctx context.Context, c *LLMConf) model.BaseModel[*einoschema.Message] {
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:      c.ApiKey,
		Model:       c.Model,
		MaxTokens:   utils.Pointer(c.MaxTokens),
		Temperature: utils.Pointer(c.Temperature),
		TopP:        utils.Pointer(c.TopP),
	})
	if err != nil {
		panic(err)
	}
	return chatModel
}
