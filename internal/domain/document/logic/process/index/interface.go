package index

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// StageSkipNone 不参与断点恢复跳过判定的阶段编码
const StageSkipNone enum.TaskStage = 0

// Context 贯穿整个构建流程的上下文对象
type Context struct {
	DocumentId          int64
	TaskId              int64
	PlanId              int64
	Document            *entity.Document
	Task                *entity.DocumentTask
	Plan                *entity.DocumentStrategyPlan
	RawFileBytes        []byte
	StartTime           time.Time
	BuildStartedTime    time.Time
	PipelineSteps       vo.DocumentStrategyStepDrafts
	ParentCandidates    vo.ParentChunkCandidates
	ChildChunks         []*entity.DocumentChunk
	ParentChunks        []*entity.DocumentParentChunk
	GraphRagBuildResult *vo.GraphRagBuildResult
}

// ShouldSkip 断点恢复判定：任务当前阶段已越过 stage 时，说明该阶段产物已就绪，恢复执行时跳过。
// stage 为 StageSkipNone 时恒不跳过（准备、完成等每次执行都必须执行的阶段）。
func (c *Context) ShouldSkip(stage enum.TaskStage) bool {
	if c == nil || c.Task == nil || stage == StageSkipNone {
		return false
	}
	return c.Task.CurrentStage > stage
}

// Stage 索引构建阶段接口
type Stage interface {
	// Name 阶段名称
	Name() string
	// TaskStage 断点恢复判定使用的任务阶段编码：任务当前阶段大于该值时，本阶段已执行过，恢复执行时跳过
	TaskStage() enum.TaskStage
	// Execute 执行阶段
	Execute(ctx context.Context, buildCtx *Context) error
}

type IndexingConfigResolver interface {
	Resolve(ctx context.Context, document *entity.Document) *vo.IndexingOptions
}

type GraphRagBuilder interface {
	RebuildDocumentGraph(ctx context.Context, documentId, taskId, knowledgeBaseId int64, chunks []*entity.DocumentChunk) (*vo.GraphRagBuildResult, error)
}

type Tokenizer interface {
	// SegmentWords 返回词文本列表，适用于搜索索引等场景
	SegmentWords(text string) []string
}

// CrossDocumentIndexer 跨文档索引器
type CrossDocumentIndexer interface {
	// RebuildAll 重建所有跨文档索引
	RebuildAll(ctx context.Context, documentId, taskId int64) error
}

// RaptorBuilder RAPTOR 层级摘要树构建器
type RaptorBuilder interface {
	// RebuildDocumentTree 重建文档树
	RebuildDocumentTree(ctx context.Context, documentId, taskId int64, chunks []*entity.DocumentChunk) (*vo.RaptorBuildResult, error)
}
