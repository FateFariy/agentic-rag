package server

import (
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/services/knowledge/api"
)

type Server struct {
	HTTP *rest.Server
	RPC  *zrpc.RpcServer // 为 nil 表示不暴露 gRPC
}

func NewServer(httpServer *rest.Server, rpcServer *zrpc.RpcServer) *Server {
	return &Server{HTTP: httpServer, RPC: rpcServer}
}

// Start 以 goroutine 启动 gRPC（阻塞），主 goroutine 启动 HTTP（阻塞），收到终止信号时由 proc 统一关停阶段优雅停止两端口
func (s *Server) Start() {
	if s.RPC != nil {
		go s.RPC.Start()
	}
	s.HTTP.Start()
}

// Stop 停止 HTTP 服务。供装配层公开 API 委托；进程信号场景下关停由 proc 统一处理
func (s *Server) Stop() {
	s.HTTP.Stop()
}

// NewHTTPServer 创建并注册 knowledge 服务的全部 HTTP 路由。
func NewHTTPServer(httpConf rest.RestConf, knowSrv api.HTTPServer) *rest.Server {
	srv := infra.NewRestServer(httpConf)
	api.RegisterHandlers(srv, knowSrv)
	return srv
}

// NewRPCServer 以无 Etcd 直连模式构造 gRPC server；register 负责注册 knowledge
// proto 服务实现。dev/test 模式下额外注册 gRPC reflection。
func NewRPCServer(c zrpc.RpcServerConf, register func(*grpc.Server)) *zrpc.RpcServer {
	return zrpc.MustNewServer(c, func(grpcServer *grpc.Server) {
		register(grpcServer)
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
}
