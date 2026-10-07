package infra

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DialGrpc 与上游微服务建立 gRPC 连接
func DialGrpc(target string) (*grpc.ClientConn, error) {
	return grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
}
