package infra

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/zeromicro/go-zero/core/logx"
)

// Neo4jConf Neo4j 图谱数据库配置
type Neo4jConf struct {
	Enabled             bool   `json:",default=false"`
	Uri                 string `json:",default=neo4j://127.0.0.1:7687"`
	Username            string `json:",default=neo4j"`
	Password            string `json:",default=neo4j"`
	Database            string `json:",default=neo4j"`
	QueryTimeoutSeconds int    `json:",default=5"`
}

// Neo4jGraphStore 基于 Neo4j 的图谱存储与写入网关（检索逻辑已下沉至 chat 服务）。
// 内嵌 neo4j.DriverWithContext，实例本身即驱动，可直接开会话、校验连通性、关闭连接。
type Neo4jGraphStore struct {
	neo4j.DriverWithContext
	db string
}

// NewNeo4jGraphStore 按配置创建驱动并自动应用约束，返回图谱存储。
// 未启用或连接失败时返回内嵌驱动为 nil 的实例（不阻断主流程），由 Available 判定。
func NewNeo4jGraphStore(c Neo4jConf) *Neo4jGraphStore {
	return &Neo4jGraphStore{DriverWithContext: newNeo4jDriver(c), db: c.Database}
}

// newNeo4jDriver 创建 Neo4j 驱动并自动应用约束；未启用或连接失败均返回 nil
func newNeo4jDriver(c Neo4jConf) neo4j.DriverWithContext {
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

// Available 报告底层驱动是否可用（图谱能力未启用或连接失败时为 false）
func (s *Neo4jGraphStore) Available() bool {
	return s != nil && s.DriverWithContext != nil
}

// ExecuteWrite 在一个写事务内执行 work，供 document 服务侧的图谱写入适配器复用
func (s *Neo4jGraphStore) ExecuteWrite(ctx context.Context, work func(tx neo4j.ManagedTransaction) error) error {
	if !s.Available() {
		return fmt.Errorf("neo4j driver is nil")
	}
	session := s.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return nil, work(tx)
	})
	return err
}

// ExecuteRead 在一个只读事务内执行 work 并返回其结果，供各服务的图谱读取适配器复用
func (s *Neo4jGraphStore) ExecuteRead(ctx context.Context, work func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	session := s.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.db})
	defer session.Close(ctx)
	return session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return work(tx)
	})
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
