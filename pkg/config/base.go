package config

import (
	"flag"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"

	"github.com/swiftbit/know-agent/pkg/ai"
	"github.com/swiftbit/know-agent/pkg/embedding"
	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/pkg/tokenize"
)

// AuthConf 鉴权配置
type AuthConf struct {
	AccessSecret string
	AccessExpire time.Duration
}

// BaseConfig 跨服务共享的基础设施与模型配置
type BaseConfig struct {
	Http       rest.RestConf           `json:",optional"`
	Rpc        zrpc.RpcServerConf      `json:",optional"`
	Auth       AuthConf                `json:",optional"`
	Mysql      infra.MySQLConf         `json:",optional"`
	Redis      infra.RedisConf         `json:",optional"`
	Minio      infra.MinioConf         `json:",optional"`
	Neo4j      infra.Neo4jConf         `json:",optional"`
	MQ         infra.MQConfig          `json:",optional"`
	Embedding  embedding.EmbeddingConf `json:",optional"`
	Gse        tokenize.GseConf        `json:",optional"`
	Milvus     infra.MilvusConf        `json:",optional"`
	ChatModel  map[string]*ai.LLMConf  `json:",optional"`
	JudgeModel map[string]*ai.LLMConf  `json:",optional"`
	Grpc       GrpcConf
}

// GrpcConf 微服务间 gRPC 出站上游地址（调用方使用）
type GrpcConf struct {
	DocumentAddr  string `json:",optional"`
	KnowledgeAddr string `json:",optional"`
}

// LoadConfig 从 -f 指定的 YAML 加载配置到 c（失败直接 panic）
func LoadConfig(c any, defaultPath string) {
	var configFile string
	flag.StringVar(&configFile, "f", defaultPath, "the config file")
	flag.Parse()
	conf.MustLoad(configFile, c, conf.UseEnv())
}
