package index

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/services/document/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic/process/chunk"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
	errorx "github.com/swiftbit/know-agent/services/document/internal/error"
)

// BuildIndexChain 阶段责任链
type BuildIndexChain struct {
	repo   adapter.DocumentRepository
	stages []Stage
}

func NewBuildIndexChain(
	repo adapter.DocumentRepository,
	storage adapter.Storage,
	vecIndexer adapter.VectorIndexer,
	keyIndexer adapter.KeywordIndexer,
	registry *chunk.Registry,
	resolver IndexingConfigResolver,
	tokenizer Tokenizer,
	builder GraphRagBuilder,
) *BuildIndexChain {
	stages := []Stage{
		NewPreparationStage(repo, storage),                    // 1. 准备阶段：加载任务、验证状态、下载原始文本内容、推进任务状态
		NewChunkingStage(repo, registry, resolver, tokenizer), // 2. 切块阶段：执行切块流水线
		NewChunkPostStage(repo),                               // 3. 切块后处理阶段：构建父子块实体并持久化
		NewVectorizeStage(repo, vecIndexer),                   // 4. 向量化阶段：批量向量化并回写状态
		NewKeywordIndexStage(repo, keyIndexer),                // 5. 关键词索引阶段：构建关键词索引
		NewGraphRagStage(repo, builder),                       // 6. GraphRAG 构建阶段：构建实体关系图谱
		NewCompletionStage(repo),                              // 7. 完成阶段：事务性更新任务/方案/文档状态
	}
	return &BuildIndexChain{repo: repo, stages: stages}
}

// Run 执行责任链
func (c *BuildIndexChain) Run(ctx context.Context, buildCtx *Context) error {
	// 断点恢复：先还原上一轮已落库的切块产物，保证被跳过的阶段之后续阶段仍有数据可用
	if err := c.restoreChunks(ctx, buildCtx); err != nil {
		return fmt.Errorf("恢复已落库分块失败: %w", err)
	}

	for _, stage := range c.stages {
		stageName := stage.Name()
		if buildCtx.ShouldSkip(stage.TaskStage()) {
			logx.Infof("[BuildIndexChain] 断点恢复跳过已执行阶段: %s, currentStage=%d(%s), documentId=%d, taskId=%d",
				stageName, buildCtx.Task.CurrentStage, enum.TaskStageName(buildCtx.Task.CurrentStage), buildCtx.DocumentId, buildCtx.TaskId)
			continue
		}
		startTime := time.Now()
		logx.Infof("[BuildIndexChain] 开始执行阶段: %s, documentId=%d, taskId=%d", stageName, buildCtx.DocumentId, buildCtx.TaskId)

		if err := stage.Execute(ctx, buildCtx); err != nil {
			if errors.Is(err, errorx.ErrGraphRagBuildFailed) {
				// 图谱构建为增强依赖：失败仅降级告警，不阻断后续阶段（其余阶段含 ResumeCommittedGraph 幂等跳过判断，完成阶段正常收尾）
				logx.Warnf("[BuildIndexChain] 阶段 %s 执行失败(降级处理，继续后续阶段): %v", stageName, err)
				continue
			}
			logx.Errorf("[BuildIndexChain] 阶段 %s 执行失败: %v", stageName, err)
			return fmt.Errorf("阶段 %s 执行失败: %w", stageName, err)
		}

		logx.Infof("[BuildIndexChain] 阶段 %s 执行成功, costMillis=%d", stageName, time.Since(startTime).Milliseconds())
	}
	return nil
}

// restoreChunks 断点恢复：任务阶段已越过切块后处理执行时，从库中恢复上一轮已落库的父子块。
// 未查到已落库分块说明切块产物尚未提交，回退任务阶段到切块执行，按首次执行重跑切块流程。
func (c *BuildIndexChain) restoreChunks(ctx context.Context, buildCtx *Context) error {
	if !buildCtx.ShouldSkip(enum.TaskStageChunkPostProcess) {
		return nil
	}
	chunks, err := c.repo.SelectChunks(ctx, map[string]any{
		"document_id": buildCtx.DocumentId,
		"task_id":     buildCtx.TaskId,
	})
	if err != nil {
		return err
	}
	if len(chunks) == 0 {
		return nil
	}
	buildCtx.ChildChunks = chunks
	logx.Infof("[BuildIndexChain] 断点恢复已落库分块: documentId=%d, taskId=%d, chunkCount=%d",
		buildCtx.DocumentId, buildCtx.TaskId, len(chunks))
	return nil
}
