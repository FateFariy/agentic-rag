package enum

// ============================================================
// StrategyStatus 策略状态
// ============================================================

type StrategyStatus = int

const (
	StrategyStatusWaitRecommend StrategyStatus = iota + 1 // 待推荐
	StrategyStatusRecommended                             // 已推荐
	StrategyStatusConfirmed                               // 已确认
	StrategyStatusExpired                                 // 已失效
)

func StrategyStatusName(status StrategyStatus) string {
	switch status {
	case StrategyStatusWaitRecommend:
		return "待推荐"
	case StrategyStatusRecommended:
		return "已推荐"
	case StrategyStatusConfirmed:
		return "已确认"
	case StrategyStatusExpired:
		return "已失效"
	default:
		return ""
	}
}

// ============================================================
// StrategyType 策略类型
// ============================================================

type StrategyType = int

const (
	StrategyTypeStructure StrategyType = iota + 1 // 结构切块
	StrategyTypeRecursive                         // 递归切块
	StrategyTypeSemantic                          // 语义切块
	StrategyTypeLLM                               // 大模型智能切块
)

func StrategyTypeName(st StrategyType) string {
	switch st {
	case StrategyTypeStructure:
		return "结构切块"
	case StrategyTypeRecursive:
		return "递归切块"
	case StrategyTypeSemantic:
		return "语义切块"
	case StrategyTypeLLM:
		return "大模型智能切块"
	default:
		return ""
	}
}

// ============================================================
// StrategyExecuteStatus 策略执行状态
// ============================================================

type StrategyExecuteStatus = int

const (
	StrategyExecuteStatusWaitExecute StrategyExecuteStatus = iota + 1
	StrategyExecuteStatusExecuting
	StrategyExecuteStatusExecuteSuccess
	StrategyExecuteStatusExecuteFailed
	StrategyExecuteStatusSkipped
)

// ============================================================
// StrategySourceType 策略来源类型
// ============================================================

type StrategySourceType = int

const (
	StrategySourceTypeSystemRecommend StrategySourceType = iota + 1
	StrategySourceTypeUserAdd
	StrategySourceTypeUserKeep
)

func StrategySourceTypeName(sourceType StrategySourceType) string {
	switch sourceType {
	case StrategySourceTypeSystemRecommend:
		return "系统推荐"
	case StrategySourceTypeUserAdd:
		return "用户添加"
	case StrategySourceTypeUserKeep:
		return "用户保留"
	default:
		return ""
	}
}
