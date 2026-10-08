package vo

import (
	"time"
)

// HybridOptions 混合检索权重配置
type HybridOptions struct {
	VectorWeight        float64
	KeywordWeight       float64
	GraphRagWeight      float64
	RankWeight          float64
	OriginalScoreWeight float64
	MetadataBoostWeight float64
	MaxMetadataBoost    float64
}

// RagRuntimeOptions RAG 运行时配置
type RagRuntimeOptions struct {
	VectorTopK                int
	KeywordTopK               int
	GraphRagTopK              int
	GraphRagMaxHops           int
	CandidateTopK             int
	RerankCandidateTopK       int
	FinalTopK                 int
	RerankEnabled             bool
	ChannelTimeout            time.Duration
	SubQuestionTimeout        time.Duration
	MinVectorSimilarity       float64
	KeywordRelativeScoreFloor float64
	KeywordChannelEnabled     bool
	GraphRagChannelEnabled    bool
	Hybrid                    *HybridOptions
	KbConfigConflictFields    []string
}

func NewDefaultHybridOptions() *HybridOptions {
	return &HybridOptions{
		VectorWeight:        0.5,
		KeywordWeight:       0.3,
		GraphRagWeight:      0.4,
		RankWeight:          0.5,
		OriginalScoreWeight: 0.7,
		MetadataBoostWeight: 0.2,
		MaxMetadataBoost:    1.0,
	}
}

func NewDefaultRagRuntimeOptions() *RagRuntimeOptions {
	return &RagRuntimeOptions{
		VectorTopK:                10,
		KeywordTopK:               10,
		GraphRagTopK:              10,
		GraphRagMaxHops:           2,
		CandidateTopK:             10,
		RerankCandidateTopK:       10,
		FinalTopK:                 10,
		RerankEnabled:             true,
		ChannelTimeout:            20 * time.Second,
		SubQuestionTimeout:        30 * time.Second,
		MinVectorSimilarity:       0.7,
		KeywordRelativeScoreFloor: 0.1,
		KeywordChannelEnabled:     true,
		GraphRagChannelEnabled:    true,
		Hybrid: &HybridOptions{
			VectorWeight:        0.5,
			KeywordWeight:       0.3,
			GraphRagWeight:      0.4,
			RankWeight:          0.5,
			OriginalScoreWeight: 0.7,
			MetadataBoostWeight: 0.2,
			MaxMetadataBoost:    1.0,
		},
		KbConfigConflictFields: []string{},
	}
}
