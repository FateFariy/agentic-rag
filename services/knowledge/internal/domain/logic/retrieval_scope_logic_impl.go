package logic

import (
	"context"
	"strconv"
	"strings"

	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/adapter"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/config"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/scope"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/aggregate"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
	errorx "github.com/swiftbit/know-agent/services/knowledge/internal/error"
)

// intentScorer 主题意图打分端口
type intentScorer interface {
	Score(ctx context.Context, question string, kbIds []int64) []*scope.NodeScore
}

// KnowledgeBaseRetrievalScopeLogicImpl 知识库检索范围
type KnowledgeBaseRetrievalScopeLogicImpl struct {
	repo          adapter.KnowledgeRepository
	docGateway    adapter.DocumentGateway
	resolver      *config.Resolver
	scorer        intentScorer
	scopeResolver *scope.RetrievalScopeResolver
}

// NewKnowledgeBaseRetrievalScopeLogicImpl 创建知识库检索范围逻辑。
// scorer 为主题意图打分器（基于本地 LLM）；传 nil 时关闭检索范围收窄，等价全库检索
func NewKnowledgeBaseRetrievalScopeLogicImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway,
	global config.GlobalConfigProvider, scorer intentScorer) *KnowledgeBaseRetrievalScopeLogicImpl {
	return &KnowledgeBaseRetrievalScopeLogicImpl{
		repo:          repo,
		docGateway:    docGateway,
		resolver:      config.NewResolver(global),
		scorer:        scorer,
		scopeResolver: scope.NewRetrievalScopeResolver(),
	}
}

// DetermineKnowledgeScope 根据知识库选择模式解析检索范围
// question 为用于意图收窄的原始问题：选库快照构建完成后，基于本地 LLM 对选中知识库的
// topic 节点打分并解析 DIRECTED/GLOBAL 范围，DIRECTED 时写入 snapshot.NarrowedDocumentIds，
// 任何收窄失败都安全降级为 GLOBAL（NarrowedDocumentIds 留空，消费方回退全库）。
func (s *KnowledgeBaseRetrievalScopeLogicImpl) DetermineKnowledgeScope(ctx context.Context, selectMode string, kbIds []string, question string) (*aggregate.KnowledgeBaseSelectionSnapshot, error) {
	// 解析选择模式
	resolvedMode := utils.BlankToDefault(selectMode, enum.KbSelectionModeNone)

	snapshot := &aggregate.KnowledgeBaseSelectionSnapshot{
		SelectionMode:     enum.KbSelectionModeNone,
		RagRuntimeOptions: s.resolver.ResolveRagRuntimeOptions(nil),
	}
	// 无选择模式时返回空快照
	if resolvedMode == enum.KbSelectionModeNone {
		return snapshot, nil
	}

	// 根据模式选择知识库
	var selectedBases []*entity.KnowledgeBase
	var err error
	switch resolvedMode {
	case enum.KbSelectionModeAll:
		selectedBases, err = s.selectAllWithRetrievableDocuments(ctx)
	case enum.KbSelectionModeSelected:
		selectedBases, err = s.selectExplicit(ctx, kbIds)
	default:
		selectedBases = nil
	}
	if err != nil {
		return nil, err
	}

	if len(selectedBases) == 0 {
		return snapshot, nil
	}

	baseIds := utils.Map(selectedBases, func(base *entity.KnowledgeBase) int64 {
		return base.ID
	})

	// 查询允许的文档
	allowedDocuments, err := s.docGateway.FindRetrievableByKbIds(ctx, baseIds)
	if err != nil {
		return nil, err
	}

	// 提取有可检索文档的知识库
	mapBy := utils.MapBy(allowedDocuments, func(doc *vo.DocumentMetadata) (int64, struct{}) {
		return doc.KnowledgeBaseId, struct{}{}
	})

	// 收集选中的知识库（仅保留有可检索文档者）
	i := 0
	for _, base := range selectedBases {
		if _, exists := mapBy[base.ID]; exists {
			selectedBases[i] = base
			i++
		}
	}
	selectedBases = selectedBases[:i]
	if len(selectedBases) == 0 {
		return snapshot, nil
	}

	// 构建快照
	snapshot.SelectionMode = resolvedMode
	snapshot.SelectedKnowledgeBases = selectedBases
	snapshot.AllowedDocuments = allowedDocuments
	// 单一配置语义：恰好选中一个知识库时解析其库级覆盖，否则沿用全局默认
	snapshot.RagRuntimeOptions = s.resolver.ResolveRagRuntimeOptions(selectedBases)

	// 在选库快照基础上追加检索范围收窄（question → topic 意图打分 → DIRECTED/GLOBAL 解析）
	s.narrowRetrievalScope(ctx, snapshot, question, allowedDocuments)

	return snapshot, nil
}

// narrowRetrievalScope 基于问题对选中知识库做意图打分并收窄文档检索范围。
// GLOBAL（含所有降级路径）下 snapshot.NarrowedDocumentIds 保持为空；DIRECTED 下写入收窄文档集合。
func (s *KnowledgeBaseRetrievalScopeLogicImpl) narrowRetrievalScope(ctx context.Context,
	snapshot *aggregate.KnowledgeBaseSelectionSnapshot, question string, allowedDocuments []*vo.DocumentMetadata) {
	// 未装配意图打分器（如本地 LLM 未配置）、问题或授权文档为空时跳过，保持 GLOBAL
	if s.scorer == nil || s.scopeResolver == nil {
		return
	}
	if strings.TrimSpace(question) == "" || len(snapshot.SelectedKnowledgeBaseIDs()) == 0 || len(allowedDocuments) == 0 {
		return
	}

	// 1. 本地 LLM 对选中知识库的 topic 节点打分
	scores := s.scorer.Score(ctx, question, snapshot.SelectedKnowledgeBaseIDs())
	if len(scores) == 0 {
		return
	}

	// 2. 加载 topic→文档绑定关系；失败降级 GLOBAL，不阻断检索
	bindings, err := s.repo.SelectTopicDocumentRelationsByKbIds(ctx, snapshot.SelectedKnowledgeBaseIDs())
	if err != nil {
		logx.Warnf("加载主题-文档关联失败，检索范围收窄降级为 GLOBAL: kbIds=%v, err=%v", snapshot.SelectedKnowledgeBaseIDs(), err)
		return
	}

	// 3. 解析收窄阈值（全局默认，允许经知识库 RetrievalConfigJson 覆盖）
	scopeConfig := s.resolver.ResolveScopeConfig(snapshot.SelectedKnowledgeBases)

	allowedDocIds := utils.Map(allowedDocuments, func(doc *vo.DocumentMetadata) int64 {
		return doc.DocumentId
	})

	// 4. 解析 DIRECTED/GLOBAL：nil 表示 GLOBAL（留空），非空为命中绑定文档（∩ 授权）+ 补充路
	narrowed := s.scopeResolver.Resolve(scores, allowedDocIds, bindings, scopeConfig)
	snapshot.NarrowedDocumentIds = narrowed
	if len(narrowed) > 0 {
		logx.Infof("检索范围收窄为 DIRECTED: kbIds=%v, allowed=%d, narrowed=%d, question='%s'",
			snapshot.SelectedKnowledgeBaseIDs(), len(allowedDocIds), len(narrowed), question)
	}
}

// selectAllWithRetrievableDocuments 选择所有启用且有可检索文档的知识库
func (s *KnowledgeBaseRetrievalScopeLogicImpl) selectAllWithRetrievableDocuments(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	enabledBases, err := s.repo.SelectKnowledgeBases(ctx)
	if err != nil {
		return nil, err
	}
	return enabledBases, nil
}

// selectExplicit 根据用户选择的ID列表选择知识库
func (s *KnowledgeBaseRetrievalScopeLogicImpl) selectExplicit(ctx context.Context, selectedKnowledgeBaseIds []string) ([]*entity.KnowledgeBase, error) {
	// 解析并去重知识库ID
	ids := utils.FilterMapUniqueLimit(selectedKnowledgeBaseIds, -1, func(rawId string) (int64, int64, bool) {
		id, _ := strconv.ParseInt(rawId, 10, 64)
		return id, id, id > 0
	})

	if len(ids) == 0 {
		return nil, errorx.ErrKnowledgeBaseMissing
	}

	// 查询知识库列表
	bases, err := s.repo.SelectKnowledgeBaseByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	// 构建ID索引
	byId := utils.MapBy(bases, func(base *entity.KnowledgeBase) (int64, *entity.KnowledgeBase) {
		return base.ID, base
	})

	// 检查所有请求的ID是否都存在
	for _, id := range ids {
		if _, exists := byId[id]; !exists {
			return nil, errorx.ErrKnowledgeBaseDisabled.Format(id)
		}
	}

	// 按原始ID顺序返回
	return utils.Map(ids, func(id int64) *entity.KnowledgeBase { return byId[id] }), nil
}
