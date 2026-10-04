package convert

import (
	"github.com/swiftbit/know-agent/services/document/api"
	dagg "github.com/swiftbit/know-agent/services/document/internal/domain/model/aggregate"
	den "github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	dvo "github.com/swiftbit/know-agent/services/document/internal/domain/model/vo"
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
	FromUploadDocumentReq(src *api.UploadDocumentReq) *den.Document
	FromConfirmStrategyReq(src *api.ConfirmStrategyReq) *dvo.DocumentStrategyConfirmCmd

	ToDocumentDetailResp(src *den.Document) *api.DocumentDetailResp
	ToDocumentDetailRespList(src []*den.Document) []*api.DocumentDetailResp
	ToKnowledgeDocumentOptionRespList(src []*dvo.DocumentMetadata) []*api.KnowledgeDocumentOptionResp
	ToQueryStrategyPlanResp(src *den.Document) *api.QueryStrategyPlanResp
	ToDocumentStrategyPlan(src *den.DocumentStrategyPlan) *api.DocumentStrategyPlan
	ToConfirmStrategyResp(plan *den.DocumentStrategyPlan) *api.ConfirmStrategyResp
	ToBuildIndexResp(src *dvo.DocumentIndexBuild) *api.BuildIndexResp
	ToDocumentChunkItemList(src []*den.DocumentChunk) []*api.DocumentChunkItem
	ToQueryDocumentChunkDetailResp(src *dagg.DocumentChunkDetail) *api.QueryDocumentChunkDetailResp
	ToQueryTaskLogsResp(src *den.DocumentTask) *api.QueryTaskLogsResp
	ToDocumentProfileResp(src *den.DocumentProfile) *api.DocumentProfileResp
	ToDocumentProfileRespList(src []*den.DocumentProfile) []*api.DocumentProfileResp

	// goverter:map . Model
	ToDocumentModel(src *den.Document) *model.Document
	// goverter:map . Model
	ToDocumentProfileModel(src *den.DocumentProfile) *model.DocumentProfile
	// goverter:map . Model
	ToDocumentTaskModel(src *den.DocumentTask) *model.DocumentTask
	// goverter:map . Model
	ToDocumentTaskLogModel(src *den.DocumentTaskLog) *model.DocumentTaskLog
	// goverter:map . Model
	ToDocumentStrategyPlanModel(src *den.DocumentStrategyPlan) *model.DocumentStrategyPlan
	// goverter:map . Model
	ToDocumentStrategyStepModel(src *den.DocumentStrategyStep) *model.DocumentStrategyStep
	ToDocumentStrategyStepModelList(src []*den.DocumentStrategyStep) []*model.DocumentStrategyStep
	// goverter:map . Model
	ToDocumentStructureNodeModel(src *den.StructureNode) *model.DocumentStructureNode
	ToDocumentStructureNodeModelList(src []*den.StructureNode) []*model.DocumentStructureNode
	// goverter:map . Model
	ToDocumentChunkModel(src *den.DocumentChunk) *model.DocumentChunk
	ToDocumentChunkModelList(src []*den.DocumentChunk) []*model.DocumentChunk
	// goverter:map . Model
	ToDocumentParentBlockModel(src *den.DocumentParentChunk) *model.DocumentParentChunk
	ToDocumentParentBlockModelList(src []*den.DocumentParentChunk) []*model.DocumentParentChunk
	// goverter:map . Model
	ToDocumentBlockModel(src *den.DocumentBlock) *model.DocumentBlock
	ToDocumentBlockModelList(src []*den.DocumentBlock) []*model.DocumentBlock
	// goverter:map . Model
	ToDocumentTableModel(src *den.DocumentTable) *model.DocumentTable
	ToDocumentTableModelList(src []*den.DocumentTable) []*model.DocumentTable
	// goverter:map . Model
	ToDocumentTableColumnModel(src *den.TableColumn) *model.DocumentTableColumn
	ToDocumentTableColumnModelList(src []*den.TableColumn) []*model.DocumentTableColumn
	// goverter:map . Model
	ToDocumentTableRowModel(src *den.TableRow) *model.DocumentTableRow
	ToDocumentTableRowModelList(src []*den.TableRow) []*model.DocumentTableRow
	// goverter:map . Model
	ToDocumentTableCellModel(src *den.TableCell) *model.DocumentTableCell
	ToDocumentTableCellModelList(src []*den.TableCell) []*model.DocumentTableCell
}
