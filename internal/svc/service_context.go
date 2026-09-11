package svc

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	einoschema "github.com/cloudwego/eino/schema"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
)

type ServiceContext struct {
	Config    *config.Config
	Minio     *minio.Client
	Db        *gorm.DB
	Rdb       *redis.Client
	RedSync   *redsync.Redsync
	ChatModel model.BaseModel[*einoschema.Message]
	Milvus    *milvusclient.Client
	Neo4j     neo4j.DriverWithContext
}

func NewServiceContext(c *config.Config) *ServiceContext {
	redisClient := NewRedisClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return &ServiceContext{
		Config:    c,
		Rdb:       redisClient,
		Db:        NewMySQLDB(c.Mysql),
		Minio:     NewMinioClient(c.Minio),
		RedSync:   NewRedSync(redisClient),
		ChatModel: NewArkChatModel(ctx, c.ChatModel["Ark"]),
		Milvus:    NewMilvusClient(ctx, c.Milvus),
		Neo4j:     NewNeo4jClient(c.Neo4j),
	}
}

// NewMySQLDB 创建 MySQL 数据库连接
func NewMySQLDB(c config.MysqlConf) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", c.Username, c.Password, c.Addr, c.DbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		//Logger:         logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		panic(err)
	}
	return db
}

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(c *config.Config) *redis.Client {
	conf := c.Redis[0]
	client := redis.NewClient(&redis.Options{
		Addr: conf.Host,
		DB:   0,
	})
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		panic(fmt.Sprintf("redis connection error: %v", err))
	}
	return client
}

// NewMinioClient 创建 Minio 客户端
func NewMinioClient(c config.MinioConf) *minio.Client {
	minioClient, err := minio.New(c.Addr, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKeyID, c.SecretAccessKey, ""),
		Secure: c.UseSSL,
	})

	if err != nil {
		panic(err)
	}
	return minioClient
}

// NewRedSync 创建 Redis 同步客户端
func NewRedSync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}

func NewArkChatModel(ctx context.Context, c *config.LLMConf) *ark.ChatModel {
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:      c.ApiKey,
		Model:       c.Model,
		MaxTokens:   utils.Pointer(c.MaxTokens),
		Temperature: utils.Pointer(c.Temperature),
		TopP:        utils.Pointer(c.TopP),
	})
	if err != nil {
		panic(err)
	}
	return chatModel
}

func NewMilvusClient(ctx context.Context, c config.MilvusConf) *milvusclient.Client {
	client, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: c.Addr,
	})
	if err != nil {
		panic(err)
	}
	return client
}

// NewNeo4jClient 创建 Neo4j 驱动并自动应用约束；未启用或连接失败均返回 nil（不阻断主流程）
func NewNeo4jClient(c config.Neo4jConf) neo4j.DriverWithContext {
	if !c.Enabled {
		return nil
	}

	// 创建驱动
	driver, err := neo4j.NewDriverWithContext(c.Uri, neo4j.BasicAuth(c.Username, c.Password, ""))
	if err != nil {
		logx.Errorf("neo4j 驱动初始化失败: %v", err)
		return nil
	}

	// 验证连通性
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = driver.VerifyConnectivity(ctx); err != nil {
		logx.Errorf("neo4j 连接不可用，已跳过图谱能力: %v", err)
		_ = driver.Close(ctx)
		return nil
	}

	// 读取约束脚本
	cypherBytes, err := os.ReadFile("etc/constraints.cypher")
	if err != nil {
		logx.Errorf("读取 Neo4j 约束文件失败: %v", err)
		return driver // 文件缺失不影响已建立的连接，直接返回 driver
	}

	// 应用约束
	execCtx, execCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer execCancel()

	session := driver.NewSession(execCtx, neo4j.SessionConfig{DatabaseName: c.Database})
	defer session.Close(execCtx)

	// Neo4j 一次查询仅允许一条语句，按分号拆分后逐条执行
	for _, stmt := range splitCypherStatements(string(cypherBytes)) {
		if _, err = session.Run(execCtx, stmt, nil); err != nil {
			logx.Errorf("应用 Neo4j 约束失败: %v", err)
			break
		}
	}

	return driver
}

// splitCypherStatements 拆分 Cypher 脚本为语句列表：剥离 // 注释行与空行，按分号切分并去掉尾部分号
func splitCypherStatements(script string) []string {
	var stmts []string
	var buf strings.Builder
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		buf.WriteString(line)
		buf.WriteString(" ")
		if !strings.HasSuffix(line, ";") {
			continue
		}
		stmts = append(stmts, strings.TrimSuffix(strings.TrimSpace(buf.String()), ";"))
		buf.Reset()
	}
	return stmts
}
