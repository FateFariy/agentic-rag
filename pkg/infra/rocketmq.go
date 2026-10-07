package infra

import (
	"context"
	"encoding/json"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/zeromicro/go-zero/core/logx"
)

// MQConfig RocketMQ 消息队列配置
type MQConfig struct {
	Addr       string `json:",default=127.0.0.1"`
	ParseTopic string `json:",default=know-agent-document"`
	IndexTopic string `json:",default=know-agent-index"`
	Retry      int    `json:",default=3"`
	Enabled    bool   `json:",default=false"`
}

type RocketMQMessageProducer struct {
	p rocketmq.Producer
}

func NewRocketMQMessageProducer(c MQConfig) *RocketMQMessageProducer {
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{c.Addr}),
		producer.WithRetry(c.Retry))
	if err != nil {
		panic(err)
	}
	return &RocketMQMessageProducer{
		p: p,
	}
}

// Start 启动生产者
func (m *RocketMQMessageProducer) Start() {
	if err := m.p.Start(); err != nil {
		panic(err)
	}
}

// Send 发送消息
func (m *RocketMQMessageProducer) Send(ctx context.Context, topic, key string, message any) error {
	messageJson, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = m.p.SendSync(ctx, primitive.NewMessage(topic, messageJson))
	return err
}

// Close 关闭生产者
func (m *RocketMQMessageProducer) Close() {
	if err := m.p.Shutdown(); err != nil {
		logx.Errorf("rocketmq producer shutdown failed: %v", err)
	}
}
