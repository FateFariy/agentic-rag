package handler

import (
	"context"
	"io"

	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/chat/api"
	"github.com/swiftbit/know-agent/services/chat/internal/convert"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/chat/internal/infrastructure/port/sse"
)

// ChatService 聊天服务实现
type ChatService struct {
	l logic.ConversationLogic
}

var _ api.HTTPServer = (*ChatService)(nil)

// NewChatService 创建聊天服务实例
func NewChatService(l logic.ConversationLogic) *ChatService {
	return &ChatService{
		l: l,
	}
}

// StreamChat 流式聊天
func (c *ChatService) StreamChat(ctx context.Context, writer io.Writer, req *api.ChatReq) error {
	sink, err := sse.NewSink(writer)
	if err != nil {
		return err
	}
	return c.l.OpenConversationStream(ctx, sink, convert.FromChatReq(req))
}

// StopConversation 停止会话
func (c *ChatService) StopConversation(ctx context.Context, req *api.ConversationIdentityReq) (*api.ConversationStopResp, error) {
	stopped, message, err := c.l.StopConversation(ctx, req.ConversationId)
	return &api.ConversationStopResp{
		ConversationId: req.ConversationId,
		Stopped:        stopped,
		Message:        message,
	}, err
}

// GetSessionDetail 获取会话详情
func (c *ChatService) GetSessionDetail(ctx context.Context, req *api.ConversationIdentityReq) (*api.ConversationSessionResp, error) {
	detail, err := c.l.GetSessionDetail(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return convert.ToConversationSessionResp(detail), err
}

// GetExchangeDetail 获取对话详情
func (c *ChatService) GetExchangeDetail(ctx context.Context, req *api.ConversationExchangeDetailQueryReq) (*api.ConversationExchangeDetailResp, error) {
	detail, stages, err := c.l.GetExchangeDetail(ctx, req.ConversationId, utils.StringToInt64(req.ExchangeId))
	if err != nil {
		return nil, err
	}
	return &api.ConversationExchangeDetailResp{
		ConversationId: req.ConversationId,
		Exchange:       convert.ToConversationExchange(detail),
		StageTraces:    convert.ToConversationStageTraces(stages),
	}, nil
}

// ListSessions 获取会话列表
func (c *ChatService) ListSessions(ctx context.Context, req *api.ConversationSessionListReq) (*api.ConversationSessionListResp, error) {
	records, total, err := c.l.ListSessions(ctx, req.PageNo, req.PageSize, enum.ToChatQueryMode(req.ChatMode), enum.ToChatTurnStatus(req.TurnStatus), req.Keyword)
	if err != nil {
		return nil, err
	}
	return &api.ConversationSessionListResp{
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		TotalPages: (total + int64(req.PageSize) - 1) / int64(req.PageSize),
		Total:      total,
		Records:    convert.ToConversationSessionRespList(records),
	}, err
}

// ResetConversation 重置会话
func (c *ChatService) ResetConversation(ctx context.Context, req *api.ConversationIdentityReq) (*api.ConversationResetResp, error) {
	reset, err := c.l.ResetConversation(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return convert.ToConversationResetResp(reset), err
}

// RebuildSummary 重建会话摘要
func (c *ChatService) RebuildSummary(ctx context.Context, req *api.ConversationIdentityReq) (*api.ConversationMemorySummaryResp, error) {
	summary, err := c.l.RebuildConversationSummary(ctx, req.ConversationId)
	if err != nil {
		return nil, err
	}
	return convert.ToConversationMemorySummaryResp(summary), nil
}

// GetRetrievalResults 获取检索结果
func (c *ChatService) GetRetrievalResults(ctx context.Context, req *api.RetrievalObserveReq) ([]*api.RetrievalResultResp, error) {
	results, err := c.l.GetRetrievalResults(ctx, req.ConversationId, utils.StringToInt64(req.ExchangeId))
	if err != nil {
		return nil, err
	}
	return convert.ToRetrievalResultRespList(results), err
}

// GetChannelExecutions 获取渠道执行结果
func (c *ChatService) GetChannelExecutions(ctx context.Context, req *api.RetrievalObserveReq) ([]*api.ChannelExecutionResp, error) {
	executions, err := c.l.GetChannelExecutions(ctx, req.ConversationId, utils.StringToInt64(req.ExchangeId))
	if err != nil {
		return nil, err
	}
	return convert.ToChannelExecutionRespList(executions), err
}
