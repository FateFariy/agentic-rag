package storage

import (
	"context"
	"strings"

	retrievermilvus "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/swiftbit/know-agent/pkg/infra"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/logic/retrieval/channel"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/enum"
	"github.com/swiftbit/know-agent/services/chat/internal/domain/model/vo"
)

type MilvusStorage struct {
	vectorRetriever  retriever.Retriever
	keywordRetriever retriever.Retriever
}

// NewMilvusStorage 初始化向量与关键词检索器
func NewMilvusStorage(c infra.MilvusConf, embedder embedding.Embedder) *MilvusStorage {
	metricType := retrievermilvus.MetricType(strings.ToUpper(c.MetricType))
	collection := c.Collection

	// 向量检索器（近似检索）
	vecRetriever, err := retrievermilvus.NewRetriever(context.Background(), &retrievermilvus.RetrieverConfig{
		ClientConfig: &milvusclient.ClientConfig{Address: c.Addr},
		Collection:   collection,
		TopK:         10,
		SearchMode:   search_mode.NewApproximate(metricType),
		Embedding:    embedder,
	})
	if err != nil {
		panic(err)
	}

	// 关键词检索器（BM25 稀疏向量）
	keyRetriever, err := retrievermilvus.NewRetriever(context.Background(), &retrievermilvus.RetrieverConfig{
		ClientConfig: &milvusclient.ClientConfig{Address: c.Addr},
		Collection:   collection,
		TopK:         10,
		SearchMode:   search_mode.NewSparse(retrievermilvus.BM25),
	})
	if err != nil {
		panic(err)
	}

	return &MilvusStorage{
		vectorRetriever:  vecRetriever,
		keywordRetriever: keyRetriever,
	}
}

// SearchByVector 执行向量语义检索
func (m *MilvusStorage) SearchByVector(ctx context.Context, query *channel.Input) ([]*vo.DocumentChunk, error) {
	docs, err := m.search(ctx, query, m.vectorRetriever)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		d.Channel = enum.RetrievalChannelVector
	}
	return docs, nil
}

// SearchByKeyword 执行关键词（BM25）检索
func (m *MilvusStorage) SearchByKeyword(ctx context.Context, query *channel.Input) ([]*vo.DocumentChunk, error) {
	docs, err := m.search(ctx, query, m.keywordRetriever)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		d.Channel = enum.RetrievalChannelKeyword
	}
	return docs, nil
}

// search 执行检索并将结果转换为领域模型
func (m *MilvusStorage) search(ctx context.Context, query *channel.Input, r retriever.Retriever) ([]*vo.DocumentChunk, error) {
	filterExpr := m.buildFilterExpr(query)
	docs, err := r.Retrieve(ctx, query.Question,
		retriever.WithTopK(query.TopK),
		retrievermilvus.WithFilter(filterExpr),
	)
	if err != nil {
		return nil, err
	}
	return m.convertToDocumentChunks(docs), nil
}

// buildFilterExpr 构建 Milvus 过滤表达式，包含文档ID、任务ID及各类路径/节点过滤条件
func (m *MilvusStorage) buildFilterExpr(query *channel.Input) string {
	var sb strings.Builder
	// 基础过滤：限定检索范围到指定的文档和任务
	sb.WriteString("document_id in ")
	sb.WriteString(utils.Join(query.DocumentIds, "[", "]", ", "))
	sb.WriteString(" AND task_id in ")
	sb.WriteString(utils.Join(query.TaskIds, "[", "]", ", "))

	if query.Filters == nil {
		return sb.String()
	}
	// 追加可选的过滤条件
	if clause := m.buildSectionFilter(query.Filters.SectionPathHints); clause != "" {
		sb.WriteString(clause)
	}
	if clause := m.buildStructureFilter(query.Filters); clause != "" {
		sb.WriteString(clause)
	}
	return sb.String()
}

// buildSectionFilter 构建 section_path 的模糊匹配过滤（OR 拼接）
func (m *MilvusStorage) buildSectionFilter(hints []string) string {
	if len(hints) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(" AND (")
	for i, hint := range hints {
		if i > 0 {
			sb.WriteString(" OR ")
		}
		sb.WriteString(`section_path like "%`)
		sb.WriteString(strings.ToLower(hint))
		sb.WriteString(`%"`)
	}
	sb.WriteString(")")
	return sb.String()
}

// buildStructureFilter 构建结构节点、规范路径和条目索引的过滤条件
func (m *MilvusStorage) buildStructureFilter(filters *channel.DocumentRetrieveFilters) string {
	if filters == nil {
		return ""
	}
	var sb strings.Builder
	if len(filters.StructureNodeIdHints) > 0 {
		sb.WriteString(" AND structure_node_id in ")
		sb.WriteString(utils.Join(filters.StructureNodeIdHints, "[", "]", ", "))
	}
	if len(filters.CanonicalPathHints) > 0 {
		sb.WriteString(" AND (")
		for i, hint := range filters.CanonicalPathHints {
			if i > 0 {
				sb.WriteString(" OR ")
			}
			sb.WriteString(`canonical_path like "%`)
			sb.WriteString(strings.ToLower(hint))
			sb.WriteString(`%"`)
		}
		sb.WriteString(")")
	}

	if len(filters.ItemIndexHints) > 0 {
		sb.WriteString(" AND item_index in ")
		ids := make([]int64, len(filters.ItemIndexHints))
		for i, v := range filters.ItemIndexHints {
			ids[i] = int64(v)
		}
		sb.WriteString(utils.Join(ids, "[", "]", ", "))
	}

	return sb.String()
}

// convertToDocumentChunks 将 Milvus 返回的 schema.Document 转换为领域模型 DocumentChunk
func (m *MilvusStorage) convertToDocumentChunks(retrievedDocs []*schema.Document) []*vo.DocumentChunk {
	return slice.Map(retrievedDocs, func(_ int, doc *schema.Document) *vo.DocumentChunk {
		meta := doc.MetaData
		return &vo.DocumentChunk{
			ID:                doc.ID,
			Content:           doc.Content,
			OriginalSnippet:   doc.Content,
			SourceType:        "DOCUMENT",
			Score:             doc.Score(),
			TaskId:            utils.Convert[int64](meta["task_id"]),
			DocumentId:        utils.Convert[int64](meta["document_id"]),
			ChunkNo:           utils.Convert[int](meta["chunk_no"]),
			ParentChunkId:     utils.Convert[int64](meta["parent_chunk_id"]),
			SectionPath:       utils.Convert[string](meta["section_path"]),
			StructureNodeId:   utils.Convert[int64](meta["structure_node_id"]),
			StructureNodeType: utils.Convert[int](meta["structure_node_type"]),
			CanonicalPath:     utils.Convert[string](meta["canonical_path"]),
			ItemIndex:         utils.Convert[int](meta["item_index"]),
		}
	})
}
