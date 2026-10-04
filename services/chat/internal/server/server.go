package server

import (
	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/pkg/httpserver"
	"github.com/swiftbit/know-agent/pkg/mq"
	"github.com/swiftbit/know-agent/pkg/svc"
	"github.com/swiftbit/know-agent/services/chat/api"
	"github.com/swiftbit/know-agent/services/chat/internal/trigger/consumer"
)

type Server struct {
	HTTP                       *rest.Server
	semanticCacheWriteConsumer *consumer.SemanticCacheWriteConsumer
	rocketMQProducer           *mq.RocketMQMessageProducer
}

func NewServer(HTTP *rest.Server, rocketMQProducer *mq.RocketMQMessageProducer, semanticCacheWriteConsumer *consumer.SemanticCacheWriteConsumer) *Server {
	return &Server{
		HTTP:                       HTTP,
		semanticCacheWriteConsumer: semanticCacheWriteConsumer,
		rocketMQProducer:           rocketMQProducer,
	}
}

func (s *Server) Start() {
	s.rocketMQProducer.Start()
	if s.semanticCacheWriteConsumer != nil {
		s.semanticCacheWriteConsumer.Start()
	}
	s.HTTP.Start()
}

func (s *Server) Stop() {
	s.rocketMQProducer.Close()
	s.HTTP.Stop()
}

// NewHTTPServer 注册 chat 服务的全部 HTTP 路由（单体遗留，chat 搬迁后删除）。
func NewHTTPServer(svcCtx *svc.ServiceContext, chatSrv api.HTTPServer) *rest.Server {
	srv := httpserver.NewRestServer(svcCtx.Config.Http)
	api.RegisterHandlers(srv, chatSrv)
	return srv
}
