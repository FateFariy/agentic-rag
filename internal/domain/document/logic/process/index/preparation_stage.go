package index

import (
	"context"
	"errors"
	"time"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// PreparationStage 准备阶段：加载策略步骤、推进任务状态
type PreparationStage struct {
	repo    adapter.DocumentRepository
	storage adapter.Storage
}

func NewPreparationStage(repo adapter.DocumentRepository, storage adapter.Storage) *PreparationStage {
	return &PreparationStage{
		repo:    repo,
		storage: storage,
	}
}

func (p *PreparationStage) Name() string {
	return "准备阶段"
}

// TaskStage 准备阶段负责加载断点恢复标记、下载原文与推进任务状态，每次恢复都需执行，不参与跳过判定
func (p *PreparationStage) TaskStage() enum.TaskStage {
	return StageSkipNone
}

func (p *PreparationStage) Execute(ctx context.Context, buildCtx *Context) error {
	sourceParseTaskId := p.requireSourceParseTaskId(ctx, buildCtx)
	if sourceParseTaskId == 0 {
		return errors.New("索引任务缺少有效且已冻结的源解析任务")
	}

	// 断点恢复且已越过切块阶段时，无需重新下载原始文件
	if !buildCtx.ShouldSkip(enum.TaskStageChunkExecute) {
		rawFileBytes, err := p.storage.DownloadObject(ctx, buildCtx.Document.ObjectName)
		if err != nil {
			return err
		}
		buildCtx.RawFileBytes = rawFileBytes
	}
	buildCtx.BuildStartedTime = time.Now()

	// 向前推进任务状态到"切块执行中"，断点恢复时不回退已完成的阶段
	buildCtx.Task.CurrentStage = max(buildCtx.Task.CurrentStage, enum.TaskStageChunkExecute)
	buildCtx.Task.TaskStatus = enum.TaskStatusRunning
	buildCtx.Task.StartTime = utils.Pointer(buildCtx.StartTime)

	// 事务性推进任务状态到"切块执行中"
	markBuildingTx := func(txCtx context.Context) error {
		document := &entity.Document{
			ID:          buildCtx.DocumentId,
			IndexStatus: enum.IndexStatusBuilding,
		}
		if err := p.repo.UpdateDocumentById(txCtx, document); err != nil {
			return err
		}
		task := &entity.DocumentTask{
			ID:           buildCtx.TaskId,
			TaskStatus:   enum.TaskStatusRunning,
			CurrentStage: buildCtx.Task.CurrentStage,
			StartTime:    utils.Pointer(buildCtx.StartTime),
		}
		return p.repo.UpdateTaskById(txCtx, task)
	}
	return p.repo.Do(ctx, markBuildingTx)
}

// requireSourceParseTaskId 获取源解析任务 ID
func (p *PreparationStage) requireSourceParseTaskId(ctx context.Context, buildCtx *Context) int64 {
	// 查询源解析任务
	var sourceParseTask *entity.DocumentTask
	indexTask := buildCtx.Task
	if indexTask != nil && indexTask.SourceParseTaskId > 0 {
		sourceParseTask, _ = p.repo.SelectTaskById(ctx, indexTask.SourceParseTaskId)
	}

	// 验证条件
	document := buildCtx.Document
	if document == nil ||
		indexTask == nil ||
		document.ID != indexTask.DocumentId ||
		indexTask.TaskType != enum.TaskTypeBuildIndex ||
		sourceParseTask == nil ||
		document.ID != sourceParseTask.DocumentId ||
		sourceParseTask.TaskType != enum.TaskTypeParseRoute ||
		sourceParseTask.TaskStatus != enum.TaskStatusSuccess {
		return 0
	}
	return sourceParseTask.ID
}
