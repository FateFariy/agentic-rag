package channel

import (
	"fmt"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval"
)

// Input 通道输入
type Input struct {
	Question          string                   `json:"question"`          // 问题
	DocumentIds       []int64                  `json:"documentIds"`       // 文档ID列表
	TaskIds           []int64                  `json:"taskIds"`           // 任务ID列表
	TopK              int                      `json:"topK"`              // 返回数量
	Filters           *DocumentRetrieveFilters `json:"filters"`           // 过滤器
	Hops              int                      `json:"hops"`              // 跳数
	Entities          []string                 `json:"entities"`          // 图谱种子实体（LLM/Agent 注入，优先于 gse 匹配）
	QueryContextHints []string                 `json:"queryContextHints"` // 查询上下文提示（暂未使用）
}

// DocumentRetrieveFilters 文档检索过滤器
type DocumentRetrieveFilters struct {
	DocumentNameHints    []string // 文档名称提示
	SectionPathHints     []string // 段落路径提示
	CanonicalPathHints   []string // 规范路径提示
	StructureNodeIdHints []int64  // 结构节点ID提示
	ItemIndexHints       []int    // 条目索引提示
	YearHints            []string // 年份提示
}

// NewInput 根据检索执行请求和渠道名称构建文档检索请求。
func NewInput(channelName string, input *retrieval.ExecutionInput) (*Input, error) {
	if input == nil {
		return nil, fmt.Errorf("ExecutionInput is required for document retrieval")
	}

	// 获取指定渠道的配置
	channel, err := input.RequireChannel(channelName)
	if err != nil {
		return nil, err
	}

	req := &Input{
		Question:          input.SubQuestion,
		TopK:              channel.TopK,
		QueryContextHints: input.ContextHints,
		DocumentIds:       input.DocumentScope,
		TaskIds:           input.TaskScope,
		Hops:              2,
	}
	if input.GraphIntent != nil {
		req.Hops = input.GraphIntent.MaxHops
		req.Entities = utils.Copy(input.GraphIntent.Entities)
	}

	// 设置过滤器
	if input.Filters != nil {
		req.Filters = &DocumentRetrieveFilters{
			DocumentNameHints:    input.Filters.DocumentNameHints,
			SectionPathHints:     input.Filters.SectionPathHints,
			YearHints:            input.Filters.YearHints,
			CanonicalPathHints:   input.Filters.CanonicalPathHints,
			StructureNodeIdHints: input.Filters.StructureNodeIdHints,
			ItemIndexHints:       input.Filters.ItemIndexHints,
		}
	}

	return req, nil
}

func (d *DocumentRetrieveFilters) IsEmpty() bool {
	if d == nil {
		return true
	}
	return len(d.DocumentNameHints) == 0 &&
		len(d.SectionPathHints) == 0 &&
		len(d.CanonicalPathHints) == 0 &&
		len(d.ItemIndexHints) == 0 &&
		len(d.YearHints) == 0
}

func (d *Input) Validate() bool {
	return d != nil && len(d.DocumentIds) > 0 && len(d.TaskIds) > 0 && utils.IsNotBlank(d.Question)
}
