package embedding

import "context"

// EmbeddingConf 嵌入模型配置
type EmbeddingConf struct {
	Model      string // 模型名称
	Dimensions int    // 嵌入维度
	BaseURL    string `json:",default=http://localhost:11434"` // Ollama 服务地址
}

type Embedding interface {
	Dimension() int
	Embedding(ctx context.Context, texts ...string) ([][]float64, error)
}
