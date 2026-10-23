package aggregate

import (
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

type KnowledgeBaseSelectionSnapshot struct {
	SelectionMode          enum.KnowledgeBaseSelectionMode `json:"selectionMode"`
	SelectedKnowledgeBases []*entity.KnowledgeBase         `json:"selectedKnowledgeBases"`
	AllowedDocuments       []*vo.DocumentMetadata          `json:"allowedDocuments"`
	RagRuntimeOptions      *vo.RagRuntimeOptions           `json:"ragRuntimeOptions,omitempty"`
	NarrowedDocumentIds    []int64                         `json:"narrowedDocumentIds,omitempty"` // 定向收窄后的文档ID集合（命中topic绑定文档 ∩ 已授权，含补充路）；
}

// SelectionModeName 返回选择模式的名称，若快照或模式为空则返回 enum.KbSelectionModeNone
func (s *KnowledgeBaseSelectionSnapshot) SelectionModeName() string {
	if s == nil || s.SelectionMode == "" {
		return enum.KbSelectionModeNone
	}
	return s.SelectionMode
}

// SelectedKnowledgeBaseIDs 从 SelectedKnowledgeBases 实时计算选中知识库ID列表
func (s *KnowledgeBaseSelectionSnapshot) SelectedKnowledgeBaseIDs() []int64 {
	if s == nil {
		return nil
	}
	return utils.Map(s.SelectedKnowledgeBases, func(base *entity.KnowledgeBase) int64 {
		return base.ID
	})
}

// SelectedKnowledgeBaseNames 从 SelectedKnowledgeBases 实时计算选中知识库名称列表
func (s *KnowledgeBaseSelectionSnapshot) SelectedKnowledgeBaseNames() []string {
	if s == nil {
		return nil
	}
	return utils.Map(s.SelectedKnowledgeBases, func(base *entity.KnowledgeBase) string {
		return base.BaseName
	})
}
