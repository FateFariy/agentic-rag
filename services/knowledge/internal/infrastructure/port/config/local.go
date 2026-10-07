package config

import (
	"github.com/swiftbit/know-agent/services/knowledge/internal/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

type LocalConfig struct {
	conf *config.Config
}

// NewLocalConfig 以全局配置构造知识库本地配置读取器
func NewLocalConfig(conf *config.Config) *LocalConfig {
	return &LocalConfig{
		conf: conf,
	}
}

func (c *LocalConfig) CurrentOptions() *vo.RagRuntimeOptions {
	opts := vo.NewDefaultRagRuntimeOptions()
	if c == nil {
		return opts
	}

	// 覆盖所有字段
	rag := c.conf.Chat.Rag
	opts.VectorTopK = rag.Vector.TopK
	opts.KeywordTopK = rag.Keyword.TopK
	opts.GraphRagTopK = rag.Graph.SearchTopK
	opts.GraphRagMaxHops = rag.Graph.SearchMaxHops
	opts.RaptorTopK = rag.Raptor.TopK
	opts.RaptorSourceChunkTopK = rag.Raptor.SourceChunkTopK
	opts.CandidateTopK = rag.FinalTopK
	opts.RerankCandidateTopK = rag.Rerank.TopN
	opts.FinalTopK = rag.FinalTopK
	opts.RerankEnabled = rag.Rerank.Enabled
	opts.ChannelTimeout = rag.ChannelTimeout
	opts.SubQuestionTimeout = rag.SubQuestionTimeout
	opts.MinVectorSimilarity = rag.Vector.MinSimilarity
	opts.KeywordRelativeScoreFloor = rag.Keyword.RelativeScoreFloor
	opts.KeywordChannelEnabled = rag.Keyword.Enabled
	opts.GraphRagChannelEnabled = rag.Graph.Enabled
	opts.RaptorChannelEnabled = rag.Raptor.Enabled
	hybrid := rag.Hybrid
	opts.Hybrid = &vo.HybridOptions{
		VectorWeight:        hybrid.VectorWeight,
		KeywordWeight:       hybrid.KeywordWeight,
		GraphRagWeight:      hybrid.GraphRagWeight,
		RaptorWeight:        hybrid.RaptorWeight,
		RankWeight:          hybrid.RankWeight,
		OriginalScoreWeight: hybrid.OriginalScoreWeight,
		MetadataBoostWeight: hybrid.MetadataBoostWeight,
		MaxMetadataBoost:    hybrid.MaxMetadataBoost,
	}

	return opts
}

func (c *LocalConfig) CurrentIndexingOptions() *vo.IndexingOptions {
	if c == nil || c.conf == nil {
		return &vo.IndexingOptions{}
	}

	chunk := c.conf.Chunk
	rag := c.conf.Chat.Rag

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
		Raptor: vo.RaptorBuildOptions{
			RaptorBuildEnabled:        rag.Raptor.Enabled,
			RaptorMaxClusterSize:      rag.Raptor.MaxClusterSize,
			RaptorMaxLevels:           rag.Raptor.MaxLevels,
			RaptorLlmSummaryEnabled:   rag.Raptor.LlmSummaryEnabled,
			RaptorSummaryQualityFloor: rag.Raptor.SummaryQualityFloor,
		},
	}
}
