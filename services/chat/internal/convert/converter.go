package convert

import (
	"github.com/swiftbit/know-agent/services/chat/api"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/aggregate"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/vo"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/persistence/model"
)

// goverter:converter
// goverter:output:format function
// goverter:output:file ./converter_gen.go
// goverter:useZeroValueOnPointerInconsistency
// goverter:ignoreMissing
// goverter:extend TimeToString ToChatQueryModeName Int64ToString StringToInt64
// goverter:skipCopySameType
//
//go:generate goverter gen .
type ChatConverter interface {
	FromChatReq(src *api.ChatReq) *vo.ChatCommand

	ToRetrievalResultRespList(src []*entity.ChatRetrievalResult) []*api.RetrievalResultResp
	ToConversationSessionResp(src *aggregate.ConversationArchiveRecord) *api.ConversationSessionResp
	ToConversationSessionRespList(src []*aggregate.ConversationArchiveRecord) []*api.ConversationSessionResp
	ToConversationResetResp(src *vo.ConversationReset) *api.ConversationResetResp
	// goverter:map DebugTrace | ToChatDebugTrace
	ToConversationExchange(src *entity.ChatExchange) *api.ConversationExchange
	ToConversationStageTraces(src []*entity.ChatExchangeTraceStage) []*api.ConversationTraceStage
	// goverter:map UpdateTime | TimeToStringMs
	ToConversationMemorySummaryResp(src *entity.ChatMemorySummary) *api.ConversationMemorySummaryResp
	// goverter:map StartTime | TimeToStringMs
	// goverter:map EndTime | TimeToStringMs
	ToChannelExecutionResp(src *entity.ChatChannelExecution) *api.ChannelExecutionResp
	ToChannelExecutionRespList(src []*entity.ChatChannelExecution) []*api.ChannelExecutionResp

	// goverter:map . Model
	ToChatDialogueModel(src *entity.ChatDialogue) *model.ChatDialogue
	// goverter:map . Model
	ToChatExchangeModel(src *entity.ChatExchange) *model.ChatExchange
	// goverter:map . Model
	ToChatExchangeTraceStageModel(src *entity.ChatExchangeTraceStage) *model.ChatExchangeTraceStage
	// goverter:map . Model
	ToChatMemorySummaryModel(src *entity.ChatMemorySummary) *model.ChatMemorySummary
	// goverter:map . Model
	ToChatRetrievalResultModel(src *entity.ChatRetrievalResult) *model.ChatRetrievalResult
	ToChatRetrievalResultModelList(src []*entity.ChatRetrievalResult) []*model.ChatRetrievalResult
	// goverter:map . Model
	ToChatChannelExecutionModel(src *entity.ChatChannelExecution) *model.ChatChannelExecution
	ToChatChannelExecutionModelList(src []*entity.ChatChannelExecution) []*model.ChatChannelExecution
	// goverter:map . Model
	ToChatExchangeEvalModel(src *entity.ChatExchangeEval) *model.ChatExchangeEval
	ToChatExchangeEvalModelList(src []*entity.ChatExchangeEval) []*model.ChatExchangeEval
	// goverter:map . Model
	ToChatCacheEntryModel(src *entity.ChatCacheEntry) *model.ChatCacheEntry
}
