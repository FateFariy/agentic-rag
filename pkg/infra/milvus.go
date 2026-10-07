package infra

import (
	"context"

	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// MilvusConf Milvus 向量库连接配置；由 pkg/config 内嵌复用。
type MilvusConf struct {
	Addr       string `json:",default=127.0.0.1:19530"`
	Username   string `json:",optional"`
	Password   string `json:",optional"`
	Collection string
	MetricType string `json:",default=COSINE"`
}

// NewMilvusClient 创建并返回一个 Milvus 客户端连接。
func NewMilvusClient(ctx context.Context, c MilvusConf) (*milvusclient.Client, error) {
	return milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address:  c.Addr,
		Username: c.Username,
		Password: c.Password,
	})
}
