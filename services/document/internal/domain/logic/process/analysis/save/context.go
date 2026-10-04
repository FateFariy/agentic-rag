package save

import (
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/aggregate"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
)

// Context 保存阶段上下文
type Context struct {
	DocumentId     int64
	TaskId         int64
	AnalysisResult *aggregate.AnalysisResult
	StructureNodes []*entity.StructureNode
	ParsedTextPath string
}
