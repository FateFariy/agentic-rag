package consumer

import (
	"context"
	"encoding/json"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

type ParseDocumentConsumer struct {
	l     process.AsyncProcessor
	c     rocketmq.PushConsumer
	topic string
}

func NewParseDocumentConsumer(mqAddr string, parseTopic string, l process.AsyncProcessor) *ParseDocumentConsumer {
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName("parse-document-group"),
		consumer.WithNameServer([]string{mqAddr}),
	)
	if err != nil {
		panic(err)
	}
	return &ParseDocumentConsumer{
		l:     l,
		c:     c,
		topic: parseTopic,
	}
}

func (c *ParseDocumentConsumer) Start() {
	logx.Info("启动文档解析消费者...")
	go func() {
		callback := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			var msg vo.DocumentParseRouteMessage
			for i := range msgs {
				_ = json.Unmarshal(msgs[i].Body, &msg)
				if err := c.l.HandleParseRoute(ctx, msg.DocumentId, msg.TaskId); err != nil {
					logx.Errorf("解析文档失败: %s\n", err)
					return consumer.ConsumeRetryLater, err
				}
			}

			// 返回消费成功状态
			return consumer.ConsumeSuccess, nil
		}
		err := c.c.Subscribe(c.topic, consumer.MessageSelector{}, callback)

		if err != nil {
			panic(err)
		}

		// 启动消费者，注意必须在订阅之后调用
		if err = c.c.Start(); err != nil {
			panic(err)
		}
	}()
}

// Stop 关闭底层 PushConsumer
func (c *ParseDocumentConsumer) Stop() {
	_ = c.c.Shutdown()
}
