// Package app 是 knowledge 服务对模块内其他装配方（cmd 以外的集成/评测 harness）
// 公开的唯一装配入口：服务内部实现全部位于 services/knowledge/internal，
// 外部仅能经本包在本地 gRPC server 上注册 knowledge 服务，不能直接引用其内部类型。
package app

import (
	"google.golang.org/grpc"

	"github.com/swiftbit/know-agent/pkg/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/bootstrap"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
)

// RegisterGRPC 以指定配置装配 knowledge 应用，并向 s 注册其 gRPC 服务。
// documentConn 为 knowledge 出站访问 document 服务的连接。
func RegisterGRPC(s *grpc.Server, c *config.Config, documentConn *grpc.ClientConn) {
	documentForKnowledge := gateway.NewKnowledgeDocumentClient(documentConn)
	bootstrap.Build(c, documentForKnowledge).RegisterGRPC(s)
}
