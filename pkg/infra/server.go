package infra

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// LifecycleComponent 随统一生命周期启停的消息组件
type LifecycleComponent interface {
	Start()
	Stop()
}

// Server 聚合 HTTP/gRPC 端口与可选 MQ 组件，提供单一启动/关停入口
type Server struct {
	HTTP       *rest.Server
	RPC        *zrpc.RpcServer // 为 nil 表示该服务不暴露 gRPC
	components []LifecycleComponent
}

// NewServer 可变参数顺序即启动顺序：调用方一律将 producer 放首位
func NewServer(http *rest.Server, rpc *zrpc.RpcServer, components ...LifecycleComponent) *Server {
	return &Server{HTTP: http, RPC: rpc, components: components}
}

// Start 注册统一关停监听后，按 生产者 -> 消费者 -> gRPC -> HTTP 顺序启动；
// HTTP 阻塞于主 goroutine，gRPC 阻塞于独立 goroutine；收到终止信号时由 proc 统一关停
// 关停监听按组件切片反向注册，使关停顺序为 consumers -> producer
func (s *Server) Start() {
	for i := len(s.components) - 1; i >= 0; i-- {
		proc.AddShutdownListener(s.components[i].Stop)
	}
	for _, c := range s.components {
		c.Start()
	}
	if s.RPC != nil {
		go s.RPC.Start()
	}
	s.HTTP.Start()
}

// Stop 停止全部消息组件（consumers -> producer 顺序）并停止 HTTP 服务
func (s *Server) Stop() {
	for i := len(s.components) - 1; i >= 0; i-- {
		s.components[i].Stop()
	}
	s.HTTP.Stop()
}

// NewHTTPServer 创建带统一 CORS 配置的 go-zero rest.Server，并交给 register 注册本服务路由。
func NewHTTPServer(c rest.RestConf, register func(*rest.Server)) *rest.Server {
	opts := []rest.RunOption{
		rest.WithCorsHeaders("*"),
		rest.WithCustomCors(func(header http.Header) {
			header.Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,Authorization,Token,X-Token,X-User-Id,OS,Platform, Version")
			header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
			header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		}, nil, "*"),
	}

	srv := rest.MustNewServer(c, opts...)
	register(srv)
	return srv
}

// NewRPCServer 以无 Etcd 直连模式构造 gRPC server；register 负责注册 proto 服务实现
// dev/test 模式下额外注册 gRPC reflection
func NewRPCServer(c zrpc.RpcServerConf, register func(*grpc.Server)) *zrpc.RpcServer {
	return zrpc.MustNewServer(c, func(grpcServer *grpc.Server) {
		register(grpcServer)
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
}
