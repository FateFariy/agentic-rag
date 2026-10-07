package errorx

import (
	"github.com/swiftbit/know-agent/pkg/common"
)

var (
	ErrKnowledgeBaseNotFound = common.NewBizError(40001, "知识库不存在: %d")
	ErrKnowledgeBaseDisabled = common.NewBizError(40002, "知识库不存在或已停用: %d")
	ErrKnowledgeBaseMissing  = common.NewBizError(40003, "请选择至少一个知识库")
)
