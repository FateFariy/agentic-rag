package errorx

import (
	"errors"

	"github.com/swiftbit/know-agent/common"
)

var (
	ErrDocumentIndexUnavailable = common.NewBizError(20014, "文档当前没有可用索引：%s")
	ErrDistributedLockNotFound  = common.NewBizError(20020, "分布式锁[%s]不存在")
	ErrSessionNotFound          = common.NewBizError(20021, "会话不存在: %d")
	ErrSessionRunning           = errors.New("会话已在执行中")
	ErrExchangeNotFound         = common.NewBizError(20022, "对话交互不存在: %d")
	ErrNoReadyDocuments         = common.NewBizError(20023, "没有可用的已就绪文档")
)
