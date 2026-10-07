// Package gormtx 提供与业务无关的 GORM 事务上下文管理能力，
// 供各微服务的仓储实现嵌入复用。
package infra

import (
	"context"

	"gorm.io/gorm"
)

// txCtxKey 上下文存储事务 *gorm.DB 的唯一 key 类型。
type txCtxKey struct{}

// TxManager 事务管理器，负责从上下文读取事务 DB、将事务注入上下文。
// 仓储实现通过嵌入 *TxManager 获得 Do/DBWithContext/WithinTx 能力。
type TxManager struct {
	db *gorm.DB
}

// NewTxManager 创建事务管理器。
func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// Do 开启事务并执行 fn，将事务 DB 注入 fn 收到的 context。
func (m *TxManager) Do(ctx context.Context, fn func(txCtx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txCtxKey{}, tx)
		return fn(txCtx)
	})
}

// DBWithContext 优先读取上下文内事务 DB，无事务则返回默认 DB，并绑定传入 ctx。
//
//   - 若在 Do 的回调中（ctx 已注入 tx），返回该事务 tx
//   - 否则返回 TxManager 持有的默认 db（每次操作为独立 auto-commit）
func (m *TxManager) DBWithContext(ctx context.Context) *gorm.DB {
	if v := ctx.Value(txCtxKey{}); v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx.WithContext(ctx)
		}
	}
	return m.db.WithContext(ctx)
}

// NewSession 返回基于默认 DB 的全新独立会话（NewDB:true），
// 用于构建与当前 Model 链/事务无关的裸子查询。刻意不读取上下文事务，
// 以保持与原进程内实现一致的子查询语义。
func (m *TxManager) NewSession(ctx context.Context) *gorm.DB {
	return m.db.Session(&gorm.Session{NewDB: true, Context: ctx})
}

// WithinTx 执行事务回调 fn：若上下文内已有事务则复用，否则开启新事务。
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, tx *gorm.DB) error) error {
	if v := ctx.Value(txCtxKey{}); v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			tx = tx.WithContext(ctx)
			return fn(ctx, tx)
		}
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txCtxKey{}, tx)
		return fn(txCtx, tx)
	})
}
