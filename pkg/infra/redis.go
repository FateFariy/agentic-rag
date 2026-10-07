package infra

import (
	"context"
	"fmt"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
)

// RedisConf Redis 连接配置
type RedisConf struct {
	Addr     string `json:",default=127.0.0.1:6379"`
	Username string `json:",optional"`
	Password string `json:",optional"`
	DB       int    `json:",default=0"`
}

// NewRedisClient 创建 Redis 客户端并校验连通性；失败直接 panic（启动期致命错误）
func NewRedisClient(c RedisConf) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Username: c.Username,
		Password: c.Password,
		DB:       c.DB,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		panic(fmt.Sprintf("redis connection error: %v", err))
	}
	return client
}

// NewRedSync 创建 Redis 同步客户端
func NewRedSync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}
