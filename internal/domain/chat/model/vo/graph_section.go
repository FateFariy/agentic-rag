package vo

import (
	"github.com/swiftbit/know-agent/common/utils"
)

// GraphSection 结构图节点基本信息
type GraphSection struct {
	NodeId            int64  `json:"nodeId"`
	DocumentId        int64  `json:"documentId"`
	ParseTaskId       int64  `json:"parseTaskId"`
	NodeNo            int    `json:"nodeNo"`
	Depth             int    `json:"depth"`
	ParentNodeId      int64  `json:"parentId"`
	PrevSiblingNodeId int64  `json:"prevSiblingId"`
	NextSiblingNodeId int64  `json:"nextSiblingId"`
	NodeCode          string `json:"nodeCode"`
	Title             string `json:"title"`
	AnchorText        string `json:"anchorText"`
	SectionPath       string `json:"sectionPath"`
	CanonicalPath     string `json:"canonicalPath"`
	ContentText       string `json:"contentText"`
}

// DisplayTitle 返回节点的展示标题
func (s *GraphSection) DisplayTitle() string {
	if s == nil {
		return ""
	}
	if utils.IsNotBlank(s.CanonicalPath) {
		return utils.Trim(s.SectionPath)
	}
	if utils.IsNotBlank(s.NodeCode) && utils.IsNotBlank(s.Title) {
		return utils.Trim(s.NodeCode + " " + s.Title)
	}
	return s.Title
}
