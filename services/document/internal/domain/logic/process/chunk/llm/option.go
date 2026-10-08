package llm

import (
	"github.com/swiftbit/know-agent/pkg/common"
)

type options struct {
	llmMaxChars int
	enabled     bool
}

// WithLlmMaxChars 设置LLM最大字符数
func WithLlmMaxChars(llmMaxChars int) common.Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		if llmMaxChars <= 50 {
			llmMaxChars = defaultLLMMaxChars
		}
		o.llmMaxChars = llmMaxChars
	})
}

// WithEnabled 设置是否启用LLM分块
func WithEnabled(enabled bool) common.Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		o.enabled = enabled
	})
}
