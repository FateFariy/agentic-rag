// Package errorx 定义 document 服务专属业务错误码。
// 错误码沿用单体时期 200xx 段编号，保持与历史数据/日志的一致性。
package errorx

import (
	"github.com/swiftbit/know-agent/pkg/common"
)

var (
	ErrDocumentNotFound      = common.NewBizError(20001, "文档不存在：%d")
	ErrUnsupportedFileType   = common.NewBizError(20002, "暂不支持文件类型: %s")
	ErrEmptyFileContent      = common.NewBizError(20003, "文件内容为空：%s")
	ErrDocumentStatusInvalid = common.NewBizError(20004, "文档状态无效：%s")
	ErrStrategyPlanNotFound  = common.NewBizError(20006, "策略方案不存在：%d")
	ErrStrategyStepEmpty     = common.NewBizError(20007, "当前没有可执行的策略步骤")
	ErrIndexTaskRunning      = common.NewBizError(20008, "当前文档已有索引任务正在执行")
	ErrGraphRagBuildFailed   = common.NewBizError(20012, "Graph 已构建失败")
	ErrTaskNotFound          = common.NewBizError(20016, "任务不存在：%d")
	ErrParentBlockMissing    = common.NewBizError(20017, "当前方案缺少父块流水线，无法生成 Parent-Child 结构")
	ErrChildBlockMissing     = common.NewBizError(20018, "当前方案缺少子块流水线，无法生成 Parent-Child 结构")
	ErrDocumentBlocksMissing = common.NewBizError(20019, "当前文档没有可用于切块的结构化 blocks")
)
