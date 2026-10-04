package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/vo"
)

type DistributedLock interface {
	// TryLock 尝试获取锁
	TryLock(ctx context.Context, name string) error

	// Lock 获取锁
	Lock(ctx context.Context, name string) error

	// UnlockContext 释放锁
	UnlockContext(ctx context.Context, name string) error

	// Unlock 释放锁
	Unlock(name string) error

	// Extend 锁续期
	Extend(ctx context.Context, name string) error
}

type DocumentGateway interface {
	// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据
	FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error)

	// FindRetrieveDocumentByIds 根据ID列表获取可检索的文档元数据
	FindRetrieveDocumentByIds(ctx context.Context, ids ...int64) ([]*vo.DocumentMetadata, error)

	// FindParentChunks 查询父块
	FindParentChunks(ctx context.Context, ids []int64) ([]*vo.DocumentChunk, error)
}

// Sink 面向客户端的事件输出端口
type Sink interface {
	// Text 发送文本事件
	Text(content string, conversationId string, exchangeId int64) error

	// Thinking 发送思考中事件
	Thinking(content string, conversationId string, exchangeId int64) error

	// Status 发送状态事件
	Status(content string, conversationId string, exchangeId int64) error

	// Error 发送错误事件
	Error(content string, conversationId string, exchangeId int64) error

	// References 发送参考事件
	References(references []*vo.SearchReference, conversationId string, exchangeId int64) error

	// Recommendations 发送推荐事件
	Recommendations(recommendations []string, conversationId string, exchangeId int64) error

	// Finish 发送完成事件
	Finish(conversationId string, exchangeId int64) error
}

// PromptRenderer 提示词渲染端口
type PromptRenderer interface {
	Render(templateName string, variables map[string]any) (string, error)
}

// MessageProducer 消息生产端口
type MessageProducer interface {
	Send(ctx context.Context, topic, key string, message any) error
}

type KnowledgeBaseGateway interface {
	// DetermineKnowledgeScope 确定知识范围
	DetermineKnowledgeScope(ctx context.Context, selectMode string, kbIds []string) (*vo.KnowledgeBaseSelectionSnapshot, error)
}
