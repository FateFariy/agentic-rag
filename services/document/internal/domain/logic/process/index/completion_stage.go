package index

import (
	"context"
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
)

// CompletionStage 完成阶段：事务性更新任务/方案/文档状态
type CompletionStage struct {
	repo adapter.DocumentRepository
}

func NewCompletionStage(repo adapter.DocumentRepository) *CompletionStage {
	return &CompletionStage{repo: repo}
}

func (s *CompletionStage) Name() string {
	return "完成阶段"
}

// TaskStage 完成阶段负责收尾，每次执行都必须落库最终状态，不参与跳过判定
func (s *CompletionStage) TaskStage() enum.TaskStage {
	return StageSkipNone
}

func (s *CompletionStage) Execute(ctx context.Context, buildCtx *Context) error {
	totalCostMillis := time.Since(buildCtx.BuildStartedTime).Milliseconds()

	// 事务性最终状态更新
	finalizeTx := func(txCtx context.Context) error {
		// 任务阶段推进到"存储完成"
		if err := s.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID:           buildCtx.TaskId,
			CurrentStage: enum.TaskStageStoreComplete,
		}); err != nil {
			return err
		}
		// 方案状态标记为已执行
		if err := s.repo.UpdatePlanById(txCtx, &entity.DocumentStrategyPlan{
			ID:         buildCtx.PlanId,
			PlanStatus: enum.PlanStatusExecuted,
		}); err != nil {
			return err
		}
		// 文档索引状态更新为构建成功
		if err := s.repo.UpdateDocumentById(txCtx, &entity.Document{
			ID:              buildCtx.DocumentId,
			IndexStatus:     enum.IndexStatusBuildSuccess,
			LastIndexTaskId: buildCtx.TaskId,
		}); err != nil {
			return err
		}
		// 写入成功耗时/统计
		completeTask := &entity.DocumentTask{
			ID:           buildCtx.Task.ID,
			TaskStatus:   enum.TaskStatusSuccess,
			CurrentStage: enum.TaskStageStoreComplete,
			FinishTime:   utils.Pointer(time.Now()),
			CostMillis:   time.Since(buildCtx.StartTime).Milliseconds(),
			ErrorCode:    utils.Pointer(""),
			ErrorMsg:     utils.Pointer(""),
		}
		if err := s.repo.UpdateTaskById(txCtx, completeTask); err != nil {
			return err
		}
		// 索引构建完成日志
		buildCompleteDetail, _ := json.Marshal(map[string]any{
			"parentBlockCount": len(buildCtx.ParentChunks),
			"chunkCount":       len(buildCtx.ChildChunks),
			"costMillis":       totalCostMillis,
		})
		buildCompleteLog := &entity.DocumentTaskLog{
			TaskId:       buildCtx.TaskId,
			DocumentId:   buildCtx.DocumentId,
			StageType:    enum.TaskStageStoreComplete,
			EventType:    enum.TaskEventComplete,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "索引构建完成",
			DetailJson:   string(buildCompleteDetail),
		}
		_ = s.repo.InsertTaskLog(txCtx, buildCompleteLog)

		return nil
	}
	if err := s.repo.Do(ctx, finalizeTx); err != nil {
		return err
	}

	logx.Infof("索引构建任务执行完成，documentId=%d, taskId=%d, planId=%d, parentCount=%d, chunkCount=%d, costMillis=%d",
		buildCtx.DocumentId, buildCtx.TaskId, buildCtx.PlanId,
		len(buildCtx.ParentChunks), len(buildCtx.ChildChunks), totalCostMillis)
	return nil
}
