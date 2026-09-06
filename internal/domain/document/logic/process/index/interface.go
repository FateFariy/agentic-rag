package index

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// Context 贯穿整个构建流程的上下文对象
type Context struct {
	DocumentId           int64
	TaskId               int64
	PlanId               int64
	Document             *entity.Document
	Task                 *entity.DocumentTask
	Plan                 *entity.DocumentStrategyPlan
	RawFileBytes         []byte
	StartTime            time.Time
	BuildStartedTime     time.Time
	PipelineSteps        vo.DocumentStrategyStepDrafts
	ParentCandidates     vo.ParentChunkCandidates
	ChildChunks          []*entity.DocumentChunk
	ParentChunks         []*entity.DocumentParentChunk
	GraphRagBuildResult  *vo.GraphRagBuildResult
	ResumeCommittedGraph bool
}

// Stage 索引构建阶段接口
type Stage interface {
	// Name 阶段名称
	Name() string
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
