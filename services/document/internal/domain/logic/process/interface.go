package process

import (
	"context"

	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/analysis"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
)

// AsyncProcessor 异步处理器
type AsyncProcessor interface {
	// HandleParseRoute 处理解析路由任务
	HandleParseRoute(ctx context.Context, documentId, taskId int64) error

	// HandleIndexBuild 处理索引构建任务
	HandleIndexBuild(ctx context.Context, documentId, taskId, planId int64) error
}

// ProfileGenerator 文档画像生成器
type ProfileGenerator interface {
	Generate(ctx context.Context, documentId int64, analysisResult *analysis.Result, structureNodes []*entity.StructureNode) (*entity.DocumentProfile, error)
}
