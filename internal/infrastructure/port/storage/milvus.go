package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	indexmilvus "github.com/cloudwego/eino-ext/components/indexer/milvus2"
	retrievermilvus "github.com/cloudwego/eino-ext/components/retriever/milvus2"
	"github.com/cloudwego/eino-ext/components/retriever/milvus2/search_mode"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	docenum "github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/svc"
)

// MilvusStorage 支持向量检索、关键词检索（BM25）
type MilvusStorage struct {
	client           *milvusclient.Client // Milvus 客户端，用于删除等操作
	vectorRetriever  retriever.Retriever  // 向量检索器（Dense Vector）
	keywordRetriever retriever.Retriever  // 关键词检索器（Sparse BM25）
	indexer          indexer.Indexer      // 向量索引器，用于写入 embedding
	model            string               // 当前使用的 embedding 模型名称
	collection       string               // Milvus 集合名
}

// NewMilvusStorage 构造函数：初始化所有 Milvus 组件
func NewMilvusStorage(svcCtx *svc.ServiceContext, embedder embedding.Embedder) *MilvusStorage {
	metricType := retrievermilvus.MetricType(strings.ToUpper(svcCtx.Config.Milvus.MetricType))
	collection := svcCtx.Config.Milvus.Collection

	// 初始化向量检索器（近似检索）
	vecRetriever, err := retrievermilvus.NewRetriever(context.Background(), &retrievermilvus.RetrieverConfig{
		ClientConfig: &milvusclient.ClientConfig{Address: svcCtx.Config.Milvus.Addr},
		Collection:   collection,
		TopK:         10,
		SearchMode:   search_mode.NewApproximate(metricType),
		Embedding:    embedder,
	})
	if err != nil {
		panic(err)
	}

	// 初始化关键词检索器（BM25 稀疏向量）
	keyRetriever, err := retrievermilvus.NewRetriever(context.Background(), &retrievermilvus.RetrieverConfig{
		ClientConfig: &milvusclient.ClientConfig{Address: svcCtx.Config.Milvus.Addr},
		Collection:   collection,
		TopK:         10,
		SearchMode:   search_mode.NewSparse(retrievermilvus.BM25),
	})
	if err != nil {
		panic(err)
	}

	// 初始化向量索引器（用于写入）
	indexerConfig := &indexmilvus.IndexerConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: svcCtx.Config.Milvus.Addr,
		},
		Collection: svcCtx.Config.Milvus.Collection,
		Vector: &indexmilvus.VectorConfig{
			Dimension:    int64(svcCtx.Config.Embedding.Dimensions),
			MetricType:   indexmilvus.MetricType(strings.ToUpper(svcCtx.Config.Milvus.MetricType)),
			IndexBuilder: indexmilvus.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200),
		},
		Sparse:    &indexmilvus.SparseVectorConfig{},
		Embedding: embedder,
	}
	indexerConfig.DocumentConverter = DocumentConverter(indexerConfig.Vector, indexerConfig.Sparse)
	vecIndexer, err := indexmilvus.NewIndexer(context.Background(), indexerConfig)
	if err != nil {
		panic(err)
	}

	return &MilvusStorage{
		client:           svcCtx.Milvus,
		vectorRetriever:  vecRetriever,
		keywordRetriever: keyRetriever,
		indexer:          vecIndexer,
		model:            svcCtx.Config.Embedding.Model,
		collection:       collection,
	}
}

// ==================== 对外能力接口 ====================

// SearchByVector 执行向量语义检索
func (m *MilvusStorage) SearchByVector(ctx context.Context, query *channel.Input) ([]*cvo.DocumentChunk, error) {
	return m.search(ctx, query, m.vectorRetriever)
}

// SearchByKeyword 执行关键词（BM25）检索
func (m *MilvusStorage) SearchByKeyword(ctx context.Context, query *channel.Input) ([]*cvo.DocumentChunk, error) {
	return m.search(ctx, query, m.keywordRetriever)
}

// BuildVectors 将文档分片向量化并写入 Milvus
func (m *MilvusStorage) BuildVectors(ctx context.Context, chunks []*entity.DocumentChunk) error {
	docs := m.toDocument(chunks)
	if len(docs) == 0 {
		return nil
	}
	if _, err := m.indexer.Store(ctx, docs); err != nil {
		return err
	}
	m.markSuccess(chunks)
	logx.Infof("向量生成成功, chunkCount:%d, model:%s", len(docs), m.model)
	return nil
}

// BuildIndexes 构建关键字索引（Milvus 自动处理，预留接口）
func (m *MilvusStorage) BuildIndexes(ctx context.Context, chunks []*entity.DocumentChunk) error {
	return nil
}

// DeleteByDocumentId 按文档 ID 删除 Milvus 中的相关记录
func (m *MilvusStorage) DeleteByDocumentId(ctx context.Context, documentId int64) error {
	expr := fmt.Sprintf("document_id == %d", documentId)
	_, err := m.client.Delete(ctx, milvusclient.NewDeleteOption(m.collection).WithExpr(expr))
	return err
}

// ==================== 内部通用检索逻辑 ====================

// search 通用检索方法，根据传入的 retriever 类型执行对应的检索
func (m *MilvusStorage) search(ctx context.Context, query *channel.Input, r retriever.Retriever) ([]*cvo.DocumentChunk, error) {
	filterExpr := m.buildFilterExpr(query)
	docs, err := r.Retrieve(ctx, query.Question, retriever.WithTopK(query.TopK), retrievermilvus.WithFilter(filterExpr))
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
func (m *MilvusStorage) convertToDocumentChunks(retrievedDocs []*schema.Document) []*cvo.DocumentChunk {
	return slice.Map(retrievedDocs, func(_ int, doc *schema.Document) *cvo.DocumentChunk {
		meta := doc.MetaData
		return &cvo.DocumentChunk{
			ID:                doc.ID,
			Content:           doc.Content,
			OriginalSnippet:   doc.Content,
			SourceType:        "DOCUMENT",
			Channel:           enum.RetrievalChannelVector,
			Score:             doc.Score(),
			TaskId:            utils.Convert[int64](meta[cvo.MetaTaskId]),
			DocumentId:        utils.Convert[int64](meta[cvo.MetaDocumentId]),
			ChunkNo:           utils.Convert[int](meta[cvo.MetaChunkNo]),
			ParentChunkId:     utils.Convert[int64](meta[cvo.MetaParentBlockId]),
			SectionPath:       convertor.ToString(meta[cvo.MetaSectionPath]),
			StructureNodeId:   utils.Convert[int64](meta[cvo.MetaStructureNodeId]),
			StructureNodeType: utils.Convert[int](meta[cvo.MetaStructureNodeType]),
			CanonicalPath:     convertor.ToString(meta[cvo.MetaCanonicalPath]),
			ItemIndex:         utils.Convert[int](meta[cvo.MetaItemIndex]),
		}
	})
}

// ==================== 向量写入辅助方法 ====================

// markSuccess 标记分片向量写入成功，更新状态和向量ID
func (m *MilvusStorage) markSuccess(chunks []*entity.DocumentChunk) {
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.ChunkText) {
			chunk.VectorId = strconv.FormatInt(chunk.ID, 10)
			chunk.VectorStoreType = docenum.VectorStoreTypeMilvus
			chunk.VectorStatus = docenum.VectorStatusVectorSuccess
		}
	}
}

// toDocument 将领域分片实体转换为 Milvus 写入所需的 schema.Document 格式
func (m *MilvusStorage) toDocument(chunks []*entity.DocumentChunk) []*schema.Document {
	result := make([]*schema.Document, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil && strutil.IsNotBlank(chunk.ChunkText) {
			result = append(result, &schema.Document{
				ID:      strconv.FormatInt(chunk.ID, 10),
				Content: chunk.ChunkText,
				MetaData: map[string]any{
					cvo.MetaDocumentId:        chunk.DocumentId,
					cvo.MetaTaskId:            chunk.TaskId,
					cvo.MetaPlanId:            chunk.PlanId,
					cvo.MetaParentBlockId:     chunk.ParentChunkId,
					cvo.MetaChunkNo:           chunk.ChunkNo,
					cvo.MetaSourceType:        chunk.SourceType,
					cvo.MetaSectionPath:       chunk.SectionPath,
					cvo.MetaStructureNodeId:   chunk.StructureNodeId,
					cvo.MetaStructureNodeType: chunk.StructureNodeType,
					cvo.MetaCanonicalPath:     chunk.CanonicalPath,
					cvo.MetaItemIndex:         chunk.ItemIndex,
					"charCount":               chunk.CharCount,
					"tokenCount":              chunk.TokenCount,
					"embeddingModel":          m.model,
					"createTime":              time.Now(),
					"updateTime":              time.Now(),
					"status":                  int8(0),
				},
			})
		}
	}
	return result
}
