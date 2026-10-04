package handler

import (
	"context"
	"mime/multipart"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/api"
	"github.com/swiftbit/know-agent/services/document/internal/convert"
	"github.com/swiftbit/know-agent/services/document/internal/domain/logic"
)

type DocumentService struct {
	lifeCycleLogic logic.LifecycleLogic
	profileLogic   logic.ProfileLogic
}

var _ api.HTTPServer = (*DocumentService)(nil)

func NewDocumentService(lifeCycleLogic logic.LifecycleLogic, profileLogic logic.ProfileLogic) *DocumentService {
	return &DocumentService{
		lifeCycleLogic: lifeCycleLogic,
		profileLogic:   profileLogic,
	}
}

// UploadDocument 上传文档
func (d *DocumentService) UploadDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *api.UploadDocumentReq) (*api.UploadDocumentResp, error) {
	doc := convert.FromUploadDocumentReq(req)
	doc, taskId, err := d.lifeCycleLogic.Upload(ctx, file, header, doc)
	if err != nil {
		return nil, err
	}
	return &api.UploadDocumentResp{
		DocumentId:     utils.Int64ToString(doc.ID),
		DocumentName:   doc.DocumentName,
		IndexStatus:    doc.IndexStatus,
		ParseStatus:    doc.ParseStatus,
		StrategyStatus: doc.StrategyStatus,
		TaskId:         utils.Int64ToString(taskId),
	}, nil
}

// QueryDocumentPage 查询文档分页列表
func (d *DocumentService) QueryDocumentPage(ctx context.Context, req *api.QueryDocumentPageReq) (*api.QueryDocumentPageResp, error) {
	documents, total, err := d.lifeCycleLogic.QueryDocumentPage(ctx, req.PageNo, req.PageSize, strutil.Trim(req.Keyword))
	return &api.QueryDocumentPageResp{
		PageNo:   req.PageNo,
		PageSize: req.PageSize,
		Total:    total,
		Records:  convert.ToDocumentDetailRespList(documents),
	}, err
}

// QueryDocumentDetail 查询文档详情
func (d *DocumentService) QueryDocumentDetail(ctx context.Context, req *api.QueryDocumentDetailReq) (*api.DocumentDetailResp, error) {
	detail, err := d.lifeCycleLogic.QueryDocumentDetail(ctx, utils.StringToInt64(req.DocumentId))
	return convert.ToDocumentDetailResp(detail), err
}

// GetDocumentOptions 获取知识文档选项
func (d *DocumentService) GetDocumentOptions(ctx context.Context) ([]*api.KnowledgeDocumentOptionResp, error) {
	options, err := d.lifeCycleLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeDocumentOptionRespList(options), nil
}

// DeleteDocument 删除文档
func (d *DocumentService) DeleteDocument(ctx context.Context, req *api.DeleteDocumentReq) (*api.DeleteDocumentResp, error) {
	documentName, err := d.lifeCycleLogic.DeleteDocument(ctx, utils.StringToInt64(req.DocumentId))
	return &api.DeleteDocumentResp{DocumentId: req.DocumentId, DocumentName: documentName}, err
}

// QueryStrategyPlan 查询策略计划
func (d *DocumentService) QueryStrategyPlan(ctx context.Context, req *api.QueryStrategyPlanReq) (*api.QueryStrategyPlanResp, error) {
	doc, plan, err := d.lifeCycleLogic.QueryStrategyPlan(ctx, utils.StringToInt64(req.DocumentId))
	if err != nil {
		return nil, err
	}
	resp := convert.ToQueryStrategyPlanResp(doc)
	resp.Plan = convert.ToDocumentStrategyPlan(plan)
	return resp, nil
}

// ConfirmStrategy 确认策略
func (d *DocumentService) ConfirmStrategy(ctx context.Context, req *api.ConfirmStrategyReq) (*api.ConfirmStrategyResp, error) {
	plan, doc, err := d.lifeCycleLogic.ConfirmStrategy(ctx, convert.FromConfirmStrategyReq(req))
	if err != nil {
		return nil, err
	}
	resp := convert.ToConfirmStrategyResp(plan)
	resp.StrategyStatus = doc.StrategyStatus
	resp.StrategyStatusName = doc.StrategyStatusName
	return resp, nil
}

// BuildIndex 构建索引
func (d *DocumentService) BuildIndex(ctx context.Context, req *api.BuildIndexReq) (*api.BuildIndexResp, error) {
	resp, err := d.lifeCycleLogic.BuildIndex(ctx, utils.StringToInt64(req.DocumentId), utils.StringToInt64(req.PlanId), utils.StringToInt64(req.OperatorId))
	return convert.ToBuildIndexResp(resp), err
}

// QueryDocumentChunks 查询文档块列表
func (d *DocumentService) QueryDocumentChunks(ctx context.Context, req *api.QueryDocumentChunksReq) (*api.QueryDocumentChunksResp, error) {
	chunks, total, planId, err := d.lifeCycleLogic.QueryDocumentChunks(ctx, utils.StringToInt64(req.DocumentId), utils.StringToInt64(req.TaskId), req.PageNo, req.PageSize)
	taskId := int64(0)
	if total > 0 {
		taskId = chunks[0].TaskId
	}
	return &api.QueryDocumentChunksResp{
		DocumentId: req.DocumentId,
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		PlanId:     utils.Int64ToString(planId),
		Records:    convert.ToDocumentChunkItemList(chunks),
		TaskId:     utils.Int64ToString(taskId),
		Total:      total,
	}, err
}

// QueryDocumentChunkDetail 查询文档块详情
func (d *DocumentService) QueryDocumentChunkDetail(ctx context.Context, req *api.QueryDocumentChunkDetailReq) (*api.QueryDocumentChunkDetailResp, error) {
	detail, err := d.lifeCycleLogic.QueryDocumentChunkDetail(ctx, utils.StringToInt64(req.DocumentId), utils.StringToInt64(req.TaskId), utils.StringToInt64(req.ChunkId))
	return convert.ToQueryDocumentChunkDetailResp(detail), err
}

// QueryTaskLogs 查询任务日志
func (d *DocumentService) QueryTaskLogs(ctx context.Context, req *api.QueryTaskLogsReq) (*api.QueryTaskLogsResp, error) {
	task, total, err := d.lifeCycleLogic.QueryTaskLogs(ctx, utils.StringToInt64(req.TaskId), req.PageNo, req.PageSize)
	if err != nil {
		return nil, err
	}
	queryTaskLogsResp := convert.ToQueryTaskLogsResp(task)
	queryTaskLogsResp.Total = total
	return queryTaskLogsResp, nil
}

// GetDocumentProfile 查询文档画像详情
func (d *DocumentService) GetDocumentProfile(ctx context.Context, req *api.DocumentProfileDetailReq) (*api.DocumentProfileResp, error) {
	profile, err := d.profileLogic.GetProfileByDocumentId(ctx, utils.StringToInt64(req.DocumentId))
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileResp(profile), nil
}

// RegenerateDocumentProfile 重新生成文档画像
func (d *DocumentService) RegenerateDocumentProfile(ctx context.Context, req *api.DocumentProfileRegenerateReq) (*api.DocumentProfileResp, error) {
	profile, err := d.profileLogic.RegenerateProfile(ctx, utils.StringToInt64(req.DocumentId))
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileResp(profile), nil
}

// BatchRegenerateDocumentProfile 批量重新生成文档画像
func (d *DocumentService) BatchRegenerateDocumentProfile(ctx context.Context, req *api.DocumentProfileBatchRegenerateReq) ([]*api.DocumentProfileResp, error) {
	ids := slice.Map(req.DocumentIds, func(_ int, item string) int64 { return utils.StringToInt64(item) })
	profiles, err := d.profileLogic.BatchRegenerateProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileRespList(profiles), nil
}
