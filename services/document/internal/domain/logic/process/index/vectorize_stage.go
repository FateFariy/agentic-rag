package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/services/document/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
)

const embeddingBatch = 100 // 默认向量化批大小

// VectorizeStage 向量化阶段：批量向量化并回写状态
type VectorizeStage struct {
	repo    adapter.DocumentRepository
	indexer adapter.VectorIndexer
}

func NewVectorizeStage(repo adapter.DocumentRepository, indexer adapter.VectorIndexer) *VectorizeStage {
	return &VectorizeStage{
		repo:    repo,
		indexer: indexer,
	}
}

func (p *VectorizeStage) Name() string {
	return "向量化阶段"
}

// TaskStage 向量化阶段的任务阶段编码：任务阶段越过向量化说明子块向量已写入
func (p *VectorizeStage) TaskStage() enum.TaskStage {
	return enum.TaskStageVectorize
}

func (p *VectorizeStage) Execute(ctx context.Context, buildCtx *Context) error {
	buildCtx.Task.CurrentStage = enum.TaskStageVectorize

	task := &entity.DocumentTask{
		ID:           buildCtx.TaskId,
		CurrentStage: enum.TaskStageVectorize,
	}
	if err := p.repo.UpdateTaskById(ctx, task); err != nil {
		return err
	}

	vectorSize := len(buildCtx.ChildChunks)
	vectorBatch := (vectorSize + embeddingBatch - 1) / embeddingBatch
	detail := map[string]any{
		"chunkCount":          vectorSize,
		"embeddingBatchSize":  embeddingBatch,
		"embeddingBatchCount": vectorBatch,
		"vectorStoreType":     enum.VectorStoreTypeMilvus,
		"parentCount":         len(buildCtx.ParentChunks),
	}
	// 记录"开始执行向量化"日志
	vectorStartDetail, _ := json.Marshal(detail)
	vectorStartLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageVectorize,
		EventType:    enum.TaskEventStart,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "开始执行向量化",
		DetailJson:   string(vectorStartDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, vectorStartLog)
	_ = p.indexer.DeleteByDocumentId(ctx, buildCtx.DocumentId)

	// 分批向量化
	vectorStartedTime := time.Now()
	for i := 0; i < vectorSize; i += embeddingBatch {
		end := min(i+embeddingBatch, vectorSize)
		if err := p.indexer.BuildVectors(ctx, buildCtx.ChildChunks[i:end]); err != nil {
			return err
		}
	}

	// 回写向量状态
	if err := p.repo.UpdateBatchChunkById(ctx, buildCtx.ChildChunks, "vector_id"); err != nil {
		return err
	}
	// 记录"向量化完成"日志
	detail["vectorCostMillis"] = time.Since(vectorStartedTime).Milliseconds()
	vectorEndDetail, _ := json.Marshal(detail)
	vectorEndLog := &entity.DocumentTaskLog{
		TaskId:       buildCtx.TaskId,
		DocumentId:   buildCtx.DocumentId,
		StageType:    enum.TaskStageVectorize,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: enum.OperatorTypeSystem,
		Content:      "向量化完成",
		DetailJson:   string(vectorEndDetail),
	}
	_ = p.repo.InsertTaskLog(ctx, vectorEndLog)

	return nil
}
