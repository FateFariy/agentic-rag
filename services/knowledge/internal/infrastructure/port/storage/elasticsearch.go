package storage

import (
	"context"

	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/logic/route/rank"
	"github.com/swiftbit/know-agent/services/knowledge/internal/domain/model/vo"
)

type ElasticStorage struct {
}

func NewElasticStorage() *ElasticStorage {
	return &ElasticStorage{}
}

func (e *ElasticStorage) Search(ctx context.Context, input *rank.EsSearchInput) ([]*vo.RouteLexicalHit, error) {
	// 实现逻辑
	return nil, nil
}
