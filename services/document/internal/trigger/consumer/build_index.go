package consumer

import (
	"context"
	"encoding/json"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/rlog"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
)

func init() {
	rlog.SetLogLevel("error")
}

type BuildIndexConsumer struct {
	l     process.AsyncProcessor
	c     rocketmq.PushConsumer
	topic string
}

func NewBuildIndexConsumer(mqAddr string, indexTopic string, l process.AsyncProcessor) *BuildIndexConsumer {
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName("build-index-group"),
		consumer.WithNameServer([]string{mqAddr}),
	)
	if err != nil {
		panic(err)
	}
	return &BuildIndexConsumer{
		l:     l,
		c:     c,
		topic: indexTopic,
	}
}

func (b *BuildIndexConsumer) Start() {
	logx.Infof("启动索引构建消费者...")
	go func() {
		callback := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			var msg vo.DocumentIndexBuildMessage
			for i := range msgs {
				_ = json.Unmarshal(msgs[i].Body, &msg)
				go func() {
					if err := b.l.HandleIndexBuild(ctx, msg.DocumentId, msg.TaskId, msg.PlanId); err != nil {
						logx.Errorf("构建索引失败: %s\n", err)
					}
				}()
			}

			// 返回消费成功状态
			return consumer.ConsumeSuccess, nil
		}
		err := b.c.Subscribe(b.topic, consumer.MessageSelector{}, callback)

		if err != nil {
			panic(err)
		}

		// 启动消费者，注意必须在订阅之后调用
		if err = b.c.Start(); err != nil {
			panic(err)
		}
	}()
}

// Stop 关闭底层 PushConsumer。由聚合 Server 在统一关停阶段调用，
// 未 ACK 的消息由 broker 重新投递。
func (b *BuildIndexConsumer) Stop() {
	_ = b.c.Shutdown()
}
