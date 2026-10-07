package app

import (
	"google.golang.org/grpc"

	"github.com/swiftbit/know-agent/services/knowledge/cmd"
	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/port/gateway"
)

// RegisterGRPC 以指定配置装配 knowledge 应用，并向 s 注册其 gRPC 服务。
// documentConn 为 knowledge 出站访问 document 服务的连接。
func RegisterGRPC(s *grpc.Server, c *config.Config, documentConn *grpc.ClientConn) {
	documentForKnowledge := gateway.NewKnowledgeDocumentClient(documentConn)
	main.Build(c, documentForKnowledge).RegisterGRPC(s)
}
