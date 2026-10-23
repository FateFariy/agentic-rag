package config

import (
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

// Resolver 实现配置解析与合并
type Resolver struct {
	provider GlobalConfigProvider
}

// NewResolver 创建 Resolver 实例
func NewResolver(provider GlobalConfigProvider) *Resolver {
	return &Resolver{provider: provider}
}

// ResolveRagRuntimeOptions 根据知识库列表解析出最终的 RagRuntimeOptions。
// 单一配置语义：仅当恰好选中一个知识库时，其库级覆盖才叠加到全局默认之上；
// 选中 0 个或多个知识库时统一使用全局默认，不进行跨库合并与冲突检测。
func (r *Resolver) ResolveRagRuntimeOptions(knowledgeBases []*entity.KnowledgeBase) *vo.RagRuntimeOptions {
	options := r.provider.CurrentOptions()
	if len(knowledgeBases) != 1 {
		return options
	}
	kb := knowledgeBases[0]
	if kb == nil {
		return options
	}
	r.applySingle(options, r.parseConfig(kb))
	return options
}

// ResolveIndexingOptions 根据知识库列表解析出索引配置 todo 暂时返回全局默认
func (r *Resolver) ResolveIndexingOptions(knowledgeBases []*entity.KnowledgeBase) *vo.IndexingOptions {
	options := r.provider.CurrentIndexingOptions()
	if len(knowledgeBases) == 0 {
		return options
	}
	return options
}

// ResolveScopeConfig 解析检索范围收窄阈值：以全局默认配置为底，单一配置语义。
// 仅当恰好选中一个知识库时，才叠加该库 RetrievalConfigJson 中 scope 段的可选覆盖；
// 选中多个知识库时统一使用全局阈值，不按库顺序叠加，避免选取顺序影响行为。
func (r *Resolver) ResolveScopeConfig(knowledgeBases []*entity.KnowledgeBase) *vo.ScopeConfig {
	scope := r.provider.CurrentScopeConfig()
	if scope == nil {
		scope = vo.NewDefaultScopeConfig()
	} else {
		// Clone 避免覆盖污染 provider 持有的全局默认值
		scope = scope.Clone()
	}

	// 多库（含 0 库）统一走全局阈值，不叠加任何库级 scope 覆盖
	if len(knowledgeBases) != 1 {
		return scope
	}
	kb := knowledgeBases[0]
	if kb == nil || len(kb.RetrievalConfigJson) == 0 {
		return scope
	}
	var patch struct {
		Scope *ScopeOverrideConfig `json:"scope,omitempty"`
	}
	if err := json.Unmarshal(kb.RetrievalConfigJson, &patch); err != nil {
		logx.Warnf("知识库检索范围配置 JSON 解析失败，将忽略该段 scope 覆盖: knowledgeBaseId=%d, knowledgeBaseName=%s, err=%v",
			kb.ID, kb.BaseName, err)
		return scope
	}
	if patch.Scope != nil {
		applyScopeOverride(scope, patch.Scope)
	}
	return scope
}

// applyScopeOverride 将 source 中非 nil 的 scope 覆盖项应用到 target（指针覆盖风格）
func applyScopeOverride(target *vo.ScopeConfig, source *ScopeOverrideConfig) {
	if target == nil || source == nil {
		return
	}
	setIfNotNil(source.MinIntentScore, &target.MinIntentScore)
	setIfNotNil(source.ConfidenceThreshold, &target.ConfidenceThreshold)
	setIfNotNil(source.SupplementRatio, &target.SupplementRatio)
}

// parseConfig 解析单个知识库的多个 JSON 片段并合并
func (r *Resolver) parseConfig(kb *entity.KnowledgeBase) *RuntimeConfig {
	merged := &RuntimeConfig{
		Hybrid: &HybridConfig{
			VectorWeight:        new(float64),
			KeywordWeight:       new(float64),
			GraphRagWeight:      new(float64),
			RankWeight:          new(float64),
			OriginalScoreWeight: new(float64),
			MetadataBoostWeight: new(float64),
			MaxMetadataBoost:    new(float64),
		},
	}
	// 顺序合并，后解析的覆盖先解析的（如果有同名字段）
	r.mergeInto(merged, r.parseJSON(kb.RetrievalConfigJson, kb))
	r.mergeInto(merged, r.parseJSON(kb.GraphRagConfigJson, kb))
	return merged
}

// parseJSON 将 JSON 字符串解析为 RuntimeConfig，失败返回空配置
func (r *Resolver) parseJSON(raw json.RawMessage, kb *entity.KnowledgeBase) *RuntimeConfig {
	if len(raw) == 0 {
		return &RuntimeConfig{}
	}
	var cfg RuntimeConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		id := int64(0)
		name := ""
		if kb != nil {
			id = kb.ID
			name = kb.BaseName
		}
		logx.Warnf("知识库 RAG 配置 JSON 解析失败，将忽略该段配置: knowledgeBaseId=%d, knowledgeBaseName=%s, err=%v", id, name, err)
		return &RuntimeConfig{}
	}
	return &cfg
}

// mergeInto 将 source 中非 nil 字段复制到 target（target 原有值保留，除非 source 有值）
func (r *Resolver) mergeInto(target, source *RuntimeConfig) {
	if source == nil {
		return
	}
	copyIfNotNil(source.VectorTopK, &target.VectorTopK)
	copyIfNotNil(source.KeywordTopK, &target.KeywordTopK)
	copyIfNotNil(source.GraphRagTopK, &target.GraphRagTopK)
	copyIfNotNil(source.GraphRagMaxHops, &target.GraphRagMaxHops)
	copyIfNotNil(source.CandidateTopK, &target.CandidateTopK)
	copyIfNotNil(source.RerankCandidateTopK, &target.RerankCandidateTopK)
	copyIfNotNil(source.FinalTopK, &target.FinalTopK)
	copyIfNotNil(source.RerankEnabled, &target.RerankEnabled)
	copyIfNotNil(source.ChannelTimeout, &target.ChannelTimeout)
	copyIfNotNil(source.SubQuestionTimeout, &target.SubQuestionTimeout)
	copyIfNotNil(source.MinVectorSimilarity, &target.MinVectorSimilarity)
	copyIfNotNil(source.KeywordRelativeScoreFloor, &target.KeywordRelativeScoreFloor)
	copyIfNotNil(source.KeywordChannelEnabled, &target.KeywordChannelEnabled)
	copyIfNotNil(source.GraphRagChannelEnabled, &target.GraphRagChannelEnabled)

	if source.Hybrid != nil {
		if target.Hybrid == nil {
			target.Hybrid = &HybridConfig{}
		}
		r.mergeHybridInto(target.Hybrid, source.Hybrid)
	}
}

// mergeHybridInto 同 mergeInto 但处理 HybridConfig
func (r *Resolver) mergeHybridInto(target, source *HybridConfig) {
	copyIfNotNil(source.VectorWeight, &target.VectorWeight)
	copyIfNotNil(source.KeywordWeight, &target.KeywordWeight)
	copyIfNotNil(source.GraphRagWeight, &target.GraphRagWeight)
	copyIfNotNil(source.RankWeight, &target.RankWeight)
	copyIfNotNil(source.OriginalScoreWeight, &target.OriginalScoreWeight)
	copyIfNotNil(source.MetadataBoostWeight, &target.MetadataBoostWeight)
	copyIfNotNil(source.MaxMetadataBoost, &target.MaxMetadataBoost)
}

// applySingle 单知识库配置覆盖全局选项
func (r *Resolver) applySingle(options *vo.RagRuntimeOptions, cfg *RuntimeConfig) {
	setIfNotNil(cfg.VectorTopK, &options.VectorTopK)
	setIfNotNil(cfg.KeywordTopK, &options.KeywordTopK)
	setIfNotNil(cfg.GraphRagTopK, &options.GraphRagTopK)
	setIfNotNil(cfg.GraphRagMaxHops, &options.GraphRagMaxHops)
	setIfNotNil(cfg.CandidateTopK, &options.CandidateTopK)
	setIfNotNil(cfg.RerankCandidateTopK, &options.RerankCandidateTopK)
	setIfNotNil(cfg.FinalTopK, &options.FinalTopK)
	setIfNotNil(cfg.RerankEnabled, &options.RerankEnabled)
	setIfNotNil(cfg.ChannelTimeout, &options.ChannelTimeout)
	setIfNotNil(cfg.SubQuestionTimeout, &options.SubQuestionTimeout)
	setIfNotNil(cfg.MinVectorSimilarity, &options.MinVectorSimilarity)
	setIfNotNil(cfg.KeywordRelativeScoreFloor, &options.KeywordRelativeScoreFloor)
	setIfNotNil(cfg.KeywordChannelEnabled, &options.KeywordChannelEnabled)
	setIfNotNil(cfg.GraphRagChannelEnabled, &options.GraphRagChannelEnabled)

	if cfg.Hybrid != nil {
		if options.Hybrid == nil {
			options.Hybrid = vo.NewDefaultHybridOptions()
		}
		r.applySingleHybrid(options.Hybrid, cfg.Hybrid)
	}
}

// applySingleHybrid 单配置覆盖 HybridOptions
func (r *Resolver) applySingleHybrid(target *vo.HybridOptions, source *HybridConfig) {
	setIfNotNil(source.VectorWeight, &target.VectorWeight)
	setIfNotNil(source.KeywordWeight, &target.KeywordWeight)
	setIfNotNil(source.GraphRagWeight, &target.GraphRagWeight)
	setIfNotNil(source.RankWeight, &target.RankWeight)
	setIfNotNil(source.OriginalScoreWeight, &target.OriginalScoreWeight)
	setIfNotNil(source.MetadataBoostWeight, &target.MetadataBoostWeight)
	setIfNotNil(source.MaxMetadataBoost, &target.MaxMetadataBoost)
}

// setIfNotNil 如果 src 非 nil，将 src 指向的值写入 dst 指向的槽位。
// 要求 dst 指向的槽位已分配（例如全局默认值、ScopeConfig 的值字段）。
func setIfNotNil[T any](src *T, dst *T) {
	if src != nil {
		*dst = *src
	}
}

// copyIfNotNil 如果 src 非 nil，为 dst 槽位（指向 *T 的指针）分配新值并写入 src 指向的值；
// 用于 parseConfig 的基础结构体标量字段为 nil 的场景，避免 nil 解引用，
// 并保留“未声明字段为 nil、不覆盖目标全局默认值”的语义。
func copyIfNotNil[T any](src *T, dst **T) {
	if src == nil {
		return
	}
	v := *src
	*dst = &v
}

// ScopeOverrideConfig 检索范围收窄阈值的知识库级覆盖（指针 nil 表示不覆盖）
type ScopeOverrideConfig struct {
	MinIntentScore      *float64 `json:"minIntentScore,omitempty"`
	ConfidenceThreshold *float64 `json:"confidenceThreshold,omitempty"`
	SupplementRatio     *float64 `json:"supplementRatio,omitempty"`
}

type HybridConfig struct {
	VectorWeight        *float64 `json:"vectorWeight,omitempty"`
	KeywordWeight       *float64 `json:"keywordWeight,omitempty"`
	GraphRagWeight      *float64 `json:"graphRagWeight,omitempty"`
	RankWeight          *float64 `json:"rankWeight,omitempty"`
	OriginalScoreWeight *float64 `json:"originalScoreWeight,omitempty"`
	MetadataBoostWeight *float64 `json:"metadataBoostWeight,omitempty"`
	MaxMetadataBoost    *float64 `json:"maxMetadataBoost,omitempty"`
}

type RuntimeConfig struct {
	VectorTopK                *int           `json:"vectorTopK,omitempty"`
	KeywordTopK               *int           `json:"keywordTopK,omitempty"`
	GraphRagTopK              *int           `json:"graphRagTopK,omitempty"`
	GraphRagMaxHops           *int           `json:"graphRagMaxHops,omitempty"`
	CandidateTopK             *int           `json:"candidateTopK,omitempty"`
	RerankCandidateTopK       *int           `json:"rerankCandidateTopK,omitempty"`
	FinalTopK                 *int           `json:"finalTopK,omitempty"`
	RerankEnabled             *bool          `json:"rerankEnabled,omitempty"`
	ChannelTimeout            *time.Duration `json:"channelTimeout,omitempty"`
	SubQuestionTimeout        *time.Duration `json:"subQuestionTimeout,omitempty"`
	MinVectorSimilarity       *float64       `json:"minVectorSimilarity,omitempty"`
	KeywordRelativeScoreFloor *float64       `json:"keywordRelativeScoreFloor,omitempty"`
	KeywordChannelEnabled     *bool          `json:"keywordChannelEnabled,omitempty"`
	GraphRagChannelEnabled    *bool          `json:"graphRagChannelEnabled,omitempty"`
	Hybrid                    *HybridConfig  `json:"hybrid,omitempty"`
}
