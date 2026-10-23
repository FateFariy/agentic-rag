package config

import (
	"fmt"

	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

type LocalConfig struct {
	conf *config.Config
}

// NewLocalConfig 以全局配置构造知识库本地配置读取器
func NewLocalConfig(conf *config.Config) *LocalConfig {
	l := &LocalConfig{
		conf: conf,
	}
	// 启动期预热校验检索范围收窄阈值，非法配置直接启动失败（panic 带清晰错误）
	l.CurrentScopeConfig()
	return l
}

func (c *LocalConfig) CurrentOptions() *vo.RagRuntimeOptions {
	opts := vo.NewDefaultRagRuntimeOptions()
	if c == nil {
		return opts
	}

	// 覆盖所有字段
	rag := c.conf.Rag
	opts.VectorTopK = rag.Vector.TopK
	opts.KeywordTopK = rag.Keyword.TopK
	opts.GraphRagTopK = rag.Graph.SearchTopK
	opts.GraphRagMaxHops = rag.Graph.SearchMaxHops
	opts.CandidateTopK = rag.CandidateTopK
	opts.RerankCandidateTopK = rag.Rerank.TopN
	opts.FinalTopK = rag.FinalTopK
	opts.RerankEnabled = rag.Rerank.Enabled
	opts.ChannelTimeout = rag.ChannelTimeout
	opts.SubQuestionTimeout = rag.SubQuestionTimeout
	opts.MinVectorSimilarity = rag.Vector.MinSimilarity
	opts.KeywordRelativeScoreFloor = rag.Keyword.RelativeScoreFloor
	opts.KeywordChannelEnabled = rag.Keyword.Enabled
	opts.GraphRagChannelEnabled = rag.Graph.Enabled
	hybrid := rag.Hybrid
	opts.Hybrid = &vo.HybridOptions{
		VectorWeight:        hybrid.VectorWeight,
		KeywordWeight:       hybrid.KeywordWeight,
		GraphRagWeight:      hybrid.GraphRagWeight,
		RankWeight:          hybrid.RankWeight,
		OriginalScoreWeight: hybrid.OriginalScoreWeight,
		MetadataBoostWeight: hybrid.MetadataBoostWeight,
		MaxMetadataBoost:    hybrid.MaxMetadataBoost,
	}

	return opts
}

// CurrentScopeConfig 返回检索范围收窄阈值的全局默认值
func (c *LocalConfig) CurrentScopeConfig() *vo.ScopeConfig {
	scope := vo.NewDefaultScopeConfig()
	if c == nil || c.conf == nil {
		return scope
	}

	if sc := c.conf.Rag.Scope; sc != nil {
		scope.MinIntentScore = utils.DefaultIfZero(sc.MinIntentScore, scope.MinIntentScore)
		scope.ConfidenceThreshold = utils.DefaultIfZero(sc.ConfidenceThreshold, scope.ConfidenceThreshold)
		scope.SupplementRatio = utils.DefaultIfZero(sc.SupplementRatio, scope.SupplementRatio)
	}

	if err := scope.Validate(); err != nil {
		panic(fmt.Errorf("knowledge 全局配置 Rag.Scope 非法，服务启动失败: %w", err))
	}
	return scope
}

func (c *LocalConfig) CurrentIndexingOptions() *vo.IndexingOptions {
	if c == nil || c.conf == nil {
		return &vo.IndexingOptions{}
	}

	chunk := c.conf.Chunk
	rag := c.conf.Rag

	return &vo.IndexingOptions{
		Chunk: vo.ChunkOptions{
			ChildRecursiveMaxChars:           chunk.RecursiveMaxChars,
			ChildRecursiveOverlapChars:       chunk.RecursiveOverlapChars,
			ChildSemanticMaxChars:            chunk.SemanticMaxChars,
			ChildSemanticMinChars:            chunk.SemanticMinChars,
			ChildSemanticSimilarityThreshold: chunk.SemanticSimilarityThreshold,
			ParentChunkMaxChars:              chunk.ParentChunkMaxChars,
			ParentChunkOverlapChars:          chunk.ParentChunkOverlapChars,
			ParentSemanticMaxChars:           chunk.ParentSemanticMaxChars,
			ParentSemanticMinChars:           chunk.ParentSemanticMinChars,
		},
		GraphRag: vo.GraphRagBuildOptions{
			GraphRagBuildEnabled: rag.Graph.Enabled,
		},
	}
}
