package convert

import (
	"github.com/swiftbit/know-agent/services/knowledge/api"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/infrastructure/persistence/model"
)

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend .*
// goverter:skipCopySameType
//
//go:generate goverter gen .
type KnowledgeConverter interface {
	FromKnowledgeScopeSaveReq(src *api.KnowledgeScopeSaveReq) *entity.KnowledgeScopeNode
	FromKnowledgeTopicSaveReq(src *api.KnowledgeTopicSaveReq) *entity.KnowledgeTopicNode
	FromKnowledgeTopicDocumentRelationSaveReq(src *api.TopicDocumentRelationSaveReq) *entity.KnowledgeTopicDocumentRelation

	ToKnowledgeScopeResp(src *entity.KnowledgeScopeNode) *api.KnowledgeScopeResp
	ToKnowledgeTopicResp(src *entity.KnowledgeTopicNode) *api.KnowledgeTopicResp
	ToKnowledgeScopeRespList(src []*entity.KnowledgeScopeNode) []*api.KnowledgeScopeResp
	ToKnowledgeTopicRespList(src []*entity.KnowledgeTopicNode) []*api.KnowledgeTopicResp

	ToTopicDocumentRelationResp(src *entity.KnowledgeTopicDocumentRelation) *api.TopicDocumentRelationResp
	ToTopicDocumentRelationRespList(src []*entity.KnowledgeTopicDocumentRelation) []*api.TopicDocumentRelationResp
	ToKnowledgeBaseItemRespList(src []*entity.KnowledgeBase) []*api.KnowledgeBaseItemResp

	// goverter:map . Model
	ToKnowledgeScopeNodeModel(src *entity.KnowledgeScopeNode) *model.KnowledgeScopeNode
	// goverter:map . Model
	ToKnowledgeTopicNodeModel(src *entity.KnowledgeTopicNode) *model.KnowledgeTopicNode
	// goverter:map . Model
	ToTopicDocumentRelationModel(src *entity.KnowledgeTopicDocumentRelation) *model.KnowledgeTopicDocumentRelation
	// goverter:map Model.ID ID
	ToKnowledgeBaseEntity(src *model.KnowledgeBase) *entity.KnowledgeBase
	ToKnowledgeBaseEntities(src []*model.KnowledgeBase) []*entity.KnowledgeBase
	// goverter:map . Model
	ToKnowledgeBaseModel(src *entity.KnowledgeBase) *model.KnowledgeBase
	ToKnowledgeBaseModelList(src []*entity.KnowledgeBase) []*model.KnowledgeBase
}
