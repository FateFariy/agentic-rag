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

// BaseConfig 跨服务共享的基础设施与模型配置，由各微服务 MySQLCon 内嵌复用
type BaseConfig struct {
	Http       rest.RestConf
	Rpc        zrpc.RpcServerConf `json:",optional"`
	Auth       AuthConf
	Mysql      infra.MySQLCon
	Redis      infra.RedisConf
	Minio      infra.MinioConf
	Neo4j      infra.Neo4jConf
	MQ         infra.MQConfig
	Embedding  embedding.EmbeddingConf
	Gse        tokenize.GseConf
	Milvus     infra.MilvusConf
	ChatModel  map[string]*ai.LLMConf
	JudgeModel map[string]*ai.LLMConf
	Grpc       GrpcConf
}

// GrpcConf 微服务间 gRPC 出站上游地址（调用方使用）
type GrpcConf struct {
	DocumentAddr  string `json:",optional"` // document 出站地址（chat/knowledge 作为调用方）
	KnowledgeAddr string `json:",optional"` // knowledge 出站地址（chat/document 作为调用方）
}

// LoadConfig 从 -f 指定的 YAML 加载配置到 c（失败直接 panic），默认 etc/config-dev.yaml
// 各微服务在自身 config 包内复用，解析自身 MySQLCon 结构（只读取关注的配置段）
func LoadConfig(c any) {
	var configFile string
	flag.StringVar(&configFile, "f", "etc/config-dev.yaml", "the config file")
	flag.Parse()
	conf.MustLoad(configFile, c)
}
