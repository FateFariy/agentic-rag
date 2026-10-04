package rank

import "github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"

type Context struct {
	RoutingText              string
	QueryEmbedding           []float64
	SelectedKnowledgeBaseIds []int64
	AllowedDocumentIds       []int64
	Diagnostics              map[string]struct{}
	ScopeCandidates          []*vo.ScopeRouteCandidate
	TopicCandidates          []*vo.TopicRouteCandidate
	DocumentCandidates       []*vo.DocumentRouteCandidate
}
