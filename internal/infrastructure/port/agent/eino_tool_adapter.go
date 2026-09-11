package agent

import (
	"context"
	"encoding/json"
	"fmt"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation/tools"
)

// EinoToolAdapter 将通用 Tool 适配为 eino 的 tool.InvokableTool
type EinoToolAdapter[I, O any] struct {
	tool tools.Tool[I, O]
}

// NewEinoToolAdapter 以通用工具构造 eino 适配器
func NewEinoToolAdapter[I, O any](t tools.Tool[I, O]) *EinoToolAdapter[I, O] {
	return &EinoToolAdapter[I, O]{tool: t}
}

// Info 返回 eino 格式的工具元信息
func (t *EinoToolAdapter[I, O]) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info := t.tool.Info(ctx)
	params, err := utils.GoStruct2ParamsOneOf[I]()
	if err != nil {
		return nil, err
	}
	return &schema.ToolInfo{
		Name:        info.Name,
		Desc:        info.Description,
		ParamsOneOf: params,
	}, nil
}

// InvokableRun 以 JSON 字符串调用通用工具
func (t *EinoToolAdapter[I, O]) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	var args I
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			return "", fmt.Errorf("agent: parse tool args: %w", err)
		}
	}
	result, err := t.tool.Invoke(ctx, args)
	if err != nil {
		return "", err
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("agent: marshal tool result: %w", err)
	}

	return string(out), err
}
