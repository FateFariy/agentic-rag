package convert

import (
	"github.com/swiftbit/know-agent/services/document/api"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/aggregate"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
	"github.com/swiftbit/know-agent/services/document/internal/infrastructure/persistence/model"
)

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend TimeToString Int64ToString StringToInt64
// goverter:skipCopySameType
//
//go:generate goverter gen .
type DocumentConverter interface {
	FromUploadDocumentReq(src *api.UploadDocumentReq) *entity.Document
	FromConfirmStrategyReq(src *api.ConfirmStrategyReq) *vo.DocumentStrategyConfirmCmd

	ToDocumentDetailResp(src *entity.Document) *api.DocumentDetailResp
	ToDocumentDetailRespList(src []*entity.Document) []*api.DocumentDetailResp
	ToKnowledgeDocumentOptionRespList(src []*vo.DocumentMetadata) []*api.KnowledgeDocumentOptionResp
	ToQueryStrategyPlanResp(src *entity.Document) *api.QueryStrategyPlanResp
	ToDocumentStrategyPlan(src *entity.DocumentStrategyPlan) *api.DocumentStrategyPlan
	ToConfirmStrategyResp(plan *entity.DocumentStrategyPlan) *api.ConfirmStrategyResp
	ToBuildIndexResp(src *vo.DocumentIndexBuild) *api.BuildIndexResp
	ToDocumentChunkItemList(src []*entity.DocumentChunk) []*api.DocumentChunkItem
	ToQueryDocumentChunkDetailResp(src *aggregate.DocumentChunkDetail) *api.QueryDocumentChunkDetailResp
	ToQueryTaskLogsResp(src *entity.DocumentTask) *api.QueryTaskLogsResp
	ToDocumentProfileResp(src *entity.DocumentProfile) *api.DocumentProfileResp
	ToDocumentProfileRespList(src []*entity.DocumentProfile) []*api.DocumentProfileResp

	// goverter:map . Model
	ToDocumentModel(src *entity.Document) *model.Document
	// goverter:map . Model
	ToDocumentProfileModel(src *entity.DocumentProfile) *model.DocumentProfile
	// goverter:map . Model
	ToDocumentTaskModel(src *entity.DocumentTask) *model.DocumentTask
	// goverter:map . Model
	ToDocumentTaskLogModel(src *entity.DocumentTaskLog) *model.DocumentTaskLog
	// goverter:map . Model
	ToDocumentStrategyPlanModel(src *entity.DocumentStrategyPlan) *model.DocumentStrategyPlan
	// goverter:map . Model
	ToDocumentStrategyStepModel(src *entity.DocumentStrategyStep) *model.DocumentStrategyStep
	ToDocumentStrategyStepModelList(src []*entity.DocumentStrategyStep) []*model.DocumentStrategyStep
	// goverter:map . Model
	ToDocumentStructureNodeModel(src *entity.StructureNode) *model.DocumentStructureNode
	ToDocumentStructureNodeModelList(src []*entity.StructureNode) []*model.DocumentStructureNode
	// goverter:map . Model
	ToDocumentChunkModel(src *entity.DocumentChunk) *model.DocumentChunk
	ToDocumentChunkModelList(src []*entity.DocumentChunk) []*model.DocumentChunk
	// goverter:map . Model
	ToDocumentParentBlockModel(src *entity.DocumentParentChunk) *model.DocumentParentChunk
	ToDocumentParentBlockModelList(src []*entity.DocumentParentChunk) []*model.DocumentParentChunk
	// goverter:map . Model
	ToDocumentBlockModel(src *entity.DocumentBlock) *model.DocumentBlock
	ToDocumentBlockModelList(src []*entity.DocumentBlock) []*model.DocumentBlock
}
