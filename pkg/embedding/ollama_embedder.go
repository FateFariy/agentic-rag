package embedding

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/zeromicro/go-zero/rest/httpc"
)

// OllamaEmbedder 通过 Ollama 的 /api/embed HTTP 接口生成文本向量
// 请求格式见 https://github.com/ollama/ollama/blob/main/docs/api.md#generate-embeddings
type OllamaEmbedder struct {
	endpoint   string
	model      string
	dimensions int
}

// ollamaEmbedReq 对应 Ollama /api/embed 请求体
type ollamaEmbedReq struct {
	ContentType string   `header:"Content-Type"`
	Model       string   `json:"model"`
	Input       []string `json:"input"`
	Dimensions  int      `json:"dimensions"`
}

// ollamaEmbedResp 对应 Ollama /api/embed 响应体
type ollamaEmbedResp struct {
	Model      string      `json:"model"`
	Embeddings [][]float64 `json:"embeddings"`
}

// NewOllamaEmbedder 创建一个调用 Ollama 的 Embedder
func NewOllamaEmbedder(conf Config) *OllamaEmbedder {
	return &OllamaEmbedder{
		endpoint:   strings.TrimRight(conf.BaseURL, "/") + "/api/embed",
		model:      conf.Model,
		dimensions: conf.Dimensions,
	}
}

func (e *OllamaEmbedder) Dimension() int {
	return e.dimensions
}

func (e *OllamaEmbedder) Embedding(ctx context.Context, texts ...string) ([][]float64, error) {
	return e.EmbedStrings(ctx, texts)
}

// EmbedStrings 批量将文本转换为向量，返回顺序与输入一致
func (e *OllamaEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	resp, err := e.doEmbedRequest(ctx, texts)
	if err != nil {
		return nil, err
	}

	if len(resp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("ollama returned %d embeddings, expected %d", len(resp.Embeddings), len(texts))
	}
	return resp.Embeddings, nil
}

// doEmbedRequest 向 Ollama 发起 /api/embed 请求并解析结果
func (e *OllamaEmbedder) doEmbedRequest(ctx context.Context, texts []string) (*ollamaEmbedResp, error) {
	req := &ollamaEmbedReq{
		Model:       e.model,
		Input:       texts,
		ContentType: "application/json",
		Dimensions:  e.dimensions,
	}

	httpResp, err := httpc.Do(ctx, http.MethodPost, e.endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("call ollama embed: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	var respBody ollamaEmbedResp
	if err = httpc.ParseJsonBody(httpResp, &respBody); err != nil {
		return nil, fmt.Errorf("parse ollama embed response: %w", err)
	}
	return &respBody, nil
}
