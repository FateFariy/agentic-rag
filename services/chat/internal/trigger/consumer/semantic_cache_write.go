package consumer

import (
	"context"
	"encoding/json"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/conversation"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/entity"
)

// SemanticCacheWriteConsumer 语义缓存异步写入消费者：订阅写入主题，幂等双写（MySQL + Milvus）
type SemanticCacheWriteConsumer struct {
	store conversation.SemanticCacheStore
	c     rocketmq.PushConsumer
	topic string
}

func NewSemanticCacheWriteConsumer(store conversation.SemanticCacheStore, mqAddr string, writeTopic string) *SemanticCacheWriteConsumer {
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName("semantic-cache-write-group"),
		consumer.WithNameServer([]string{mqAddr}),
	)
	if err != nil {
		panic(err)
	}
	return &SemanticCacheWriteConsumer{
		store: store,
		c:     c,
		topic: writeTopic,
	}
}

func (c *SemanticCacheWriteConsumer) Start() {
	logx.Info("启动语义缓存写入消费者...")
	threading.GoSafe(func() {
		callback := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			for i := range msgs {
				var entry entity.ChatCacheEntry
				if err := json.Unmarshal(msgs[i].Body, &entry); err != nil {
					logx.Errorf("语义缓存写入消息反序列化失败: %v", err)
					return consumer.ConsumeRetryLater, err
				}
				if err := c.store.Put(ctx, &entry); err != nil {
					logx.Errorf("语义缓存写入消费失败: cacheId=%d, error=%v", entry.ID, err)
					return consumer.ConsumeRetryLater, err
				}
			}
			return consumer.ConsumeSuccess, nil
		}

		if err := c.c.Subscribe(c.topic, consumer.MessageSelector{}, callback); err != nil {
			panic(err)
		}
		if err := c.c.Start(); err != nil {
			panic(err)
		}
	})
}

// Stop 关闭底层 PushConsumer
func (c *SemanticCacheWriteConsumer) Stop() {
	_ = c.c.Shutdown()
}
