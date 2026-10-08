package convert

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/swiftbit/know-agent/services/chat/api"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/vo"
)

func TimeToString(t time.Time) string {
	return t.Format(time.DateTime)
}

func TimeToStringMs(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func ToChatQueryModeName(code int) string {
	return enum.ChatQueryModeName(code)
}

func ToChatDebugTrace(debugTraceJson string) *api.ChatDebugTrace {
	var debugTrace vo.ChatDebugTrace
	if err := json.Unmarshal([]byte(debugTraceJson), &debugTrace); err != nil {
		return nil
	}
	return &api.ChatDebugTrace{
		ExecutionMode:                 debugTrace.ExecutionMode,
		ChatMode:                      enum.ChatQueryModeName(debugTrace.ChatMode),
		OriginalQuestion:              debugTrace.OriginalQuestion,
		RewriteQuestion:               debugTrace.RewriteQuestion,
		NavigationDecision:            ToChatDocumentNavigationDecision(debugTrace.NavigationDecision),
		HistorySummary:                debugTrace.HistorySummary,
		LongTermSummary:               debugTrace.LongTermSummary,
		RecentHistoryTranscript:       debugTrace.RecentHistoryTranscript,
		AnswerRecentTranscript:        debugTrace.RecentQuestionTranscript,
		AnswerHistoryContext:          debugTrace.QuestionHistoryContext,
		AnswerHistoryFollowUpQuestion: debugTrace.QuestionHistoryFollowUpQuestion,
		HistoryCompressionApplied:     debugTrace.HistoryCompressionApplied,
		HistoryCoveredExchangeId:      debugTrace.HistoryCoveredExchangeId,
		HistoryCoveredExchangeCount:   debugTrace.HistoryCoveredExchangeCount,
		HistoryCompressionCount:       debugTrace.HistoryCompressionCount,
		CurrentDateText:               debugTrace.CurrentDateText,
		RequiresFreshSearch:           debugTrace.RequiresRealTimeSearch,
		RequiresCurrentDateAnchoring:  debugTrace.RequiresCurrentDateAnchoring,
		SelectedDocumentId:            debugTrace.SelectedDocumentId,
		SelectedTaskId:                debugTrace.SelectedTaskId,
		RetrievalNotes:                debugTrace.RetrievalNotes,
		UsedChannels:                  debugTrace.UsedChannels,
		ToolTraces:                    ToChatToolTraces(debugTrace.ToolTraces),
		ModelUsageTraces:              ToChatModelUsageTraces(debugTrace.ModelUsageTraces),
		LimitStats:                    ToChatLimitStats(debugTrace.LimitStats),
		RagSystemPrompt:               debugTrace.RagSystemPrompt,
		RagUserPrompt:                 debugTrace.RagUserPrompt,
		NoEvidenceReply:               debugTrace.NoEvidenceReply,
	}
}

func ToChatDocumentNavigationDecision(src *vo.DocumentNavigationDecision) *api.DocumentNavigationDecision {
	if src == nil {
		return nil
	}

	return &api.DocumentNavigationDecision{
		NavigationAction: src.NavigationAction,
		StructureAnchor:  ToChatStructureAnchor(src.StructureAnchor),
		ItemAnchor:       ToChatItemAnchor(src.ItemAnchor),
		SummaryText:      src.SummaryText,
	}
}

func ToChatStructureAnchor(src *vo.ConversationStructureAnchor) *api.ConversationStructureAnchor {
	if src == nil {
		return nil
	}
	return &api.ConversationStructureAnchor{
		RootSectionCode:   src.RootSectionCode,
		RootSectionTitle:  src.RootSectionTitle,
		TargetSectionHint: src.TargetSectionHint,
		StructureNodeId:   src.StructureNodeId,
		CanonicalPath:     src.CanonicalPath,
		ScopeMode:         src.ScopeMode,
	}
}

func ToChatItemAnchor(src *vo.ConversationItemAnchor) *api.ConversationItemAnchor {
	if src == nil {
		return nil
	}
	return &api.ConversationItemAnchor{
		ItemIndex:       src.ItemIndex,
		ItemText:        src.ItemText,
		StructureNodeId: src.StructureNodeId,
		CanonicalPath:   src.CanonicalPath,
	}
}

func ToChatToolTraces(src []*vo.ChatToolTrace) []*api.ChatToolTrace {
	if src == nil {
		return nil
	}
	result := make([]*api.ChatToolTrace, len(src))
	for i, t := range src {
		if t == nil {
			result[i] = nil
			continue
		}
		result[i] = &api.ChatToolTrace{
			ToolName:       t.ToolName,
			Status:         t.Status,
			InputSummary:   t.InputSummary,
			EffectiveInput: t.EffectiveInput,
			OutputSummary:  t.OutputSummary,
			ErrorMessage:   t.ErrorMessage,
			ReferenceCount: t.ReferenceCount,
			Topic:          t.Topic,
			DurationMs:     t.DurationMs,
		}
	}
	return result
}

func ToChatModelUsageTraces(src []*vo.ChatModelUsageTrace) []*api.ChatModelUsageTrace {
	if src == nil {
		return nil
	}
	result := make([]*api.ChatModelUsageTrace, len(src))
	for i, t := range src {
		if t == nil {
			result[i] = nil
			continue
		}
		result[i] = &api.ChatModelUsageTrace{
			StageName:        t.StageName,
			Provider:         t.Provider,
			Model:            t.Model,
			PromptTokens:     t.PromptTokens,
			CompletionTokens: t.CompletionTokens,
			TotalTokens:      t.TotalTokens,
			EstimatedCost:    t.EstimatedCost,
			DurationMs:       t.DurationMs,
			Status:           t.Status,
		}
	}
	return result
}

func ToChatLimitStats(src *vo.ChatLimitStats) *api.ChatLimitStats {
	if src == nil {
		return nil
	}
	return &api.ChatLimitStats{
		ModelCallsUsed:        src.ModelCallsUsed,
		ModelCallsRunLimit:    src.ModelCallsRunLimit,
		ModelCallsThreadLimit: src.ModelCallsThreadLimit,
		ToolCallsUsed:         src.ToolCallsUsed,
		ToolCallsRunLimit:     src.ToolCallsRunLimit,
		ToolCallsThreadLimit:  src.ToolCallsThreadLimit,
		LimitTriggered:        src.LimitTriggered,
		LimitReason:           src.LimitReason,
	}
}

func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

func StringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}
