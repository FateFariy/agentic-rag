package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	indexmilvus "github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	"github.com/swiftbit/know-agent/pkg/config"
	"github.com/swiftbit/know-agent/pkg/logx"
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/enum"
)

// MilvusIndexer document 服务的 Milvus 向量/关键字写入适配器
type MilvusIndexer struct {
	client     *milvusclient.Client // Milvus 客户端
	indexer    indexer.Indexer      // 向量索引器，用于写入 embedding
	model      string               // 当前使用的 embedding 模型名称
	collection string               // Milvus 集合名
}

// NewMilvusIndexer 构造 document 向量索引器
func NewMilvusIndexer(client *milvusclient.Client, c config.MilvusConf, embeddingDim int, embeddingModel string, embedder embedding.Embedder) *MilvusIndexer {
	indexerConfig := &indexmilvus.IndexerConfig{
		ClientConfig: &milvusclient.ClientConfig{
			Address: c.Addr,
		},
		Collection: c.Collection,
		Vector: &indexmilvus.VectorConfig{
			Dimension:    int64(embeddingDim),
			MetricType:   indexmilvus.MetricType(strings.ToUpper(c.MetricType)),
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
	return &MilvusIndexer{
		client:     client,
		indexer:    vecIndexer,
		model:      embeddingModel,
		collection: c.Collection,
	}
}

// DeleteByDocumentId 按文档 ID 删除 Milvus 中的相关记录
func (m *MilvusIndexer) DeleteByDocumentId(ctx context.Context, documentId int64) error {
	expr := fmt.Sprintf("document_id == %d", documentId)
	_, err := m.client.Delete(ctx, milvusclient.NewDeleteOption(m.collection).WithExpr(expr))
	return err
}

// BuildVectors 将文档分片向量化并写入 Milvus
func (m *MilvusIndexer) BuildVectors(ctx context.Context, chunks []*entity.DocumentChunk) error {
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
func (m *MilvusIndexer) BuildIndexes(ctx context.Context, chunks []*entity.DocumentChunk) error {
	return nil
}

// markSuccess 标记文档分片向量生成成功
func (m *MilvusIndexer) markSuccess(chunks []*entity.DocumentChunk) {
	for _, chunk := range chunks {
		if chunk != nil && utils.IsNotBlank(chunk.ChunkText) {
			chunk.VectorId = strconv.FormatInt(chunk.ID, 10)
			chunk.VectorStoreType = enum.VectorStoreTypeMilvus
			chunk.VectorStatus = enum.VectorStatusVectorSuccess
		}
	}
}

// toDocument 将文档分片转换为 schema.Document，注入向量写入所需的元数据
func (m *MilvusIndexer) toDocument(chunks []*entity.DocumentChunk) []*schema.Document {
	docs := make([]*schema.Document, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil && utils.IsNotBlank(chunk.ChunkText) {
			docs = append(docs, &schema.Document{
				ID:      strconv.FormatInt(chunk.ID, 10),
				Content: chunk.ChunkText,
				MetaData: map[string]any{
					"document_id":         chunk.DocumentId,
					"task_id":             chunk.TaskId,
					"plan_id":             chunk.PlanId,
					"parent_chunk_id":     chunk.ParentChunkId,
					"chunk_no":            chunk.ChunkNo,
					"source_type":         chunk.SourceType,
					"section_path":        chunk.SectionPath,
					"structure_node_id":   chunk.StructureNodeId,
					"structure_node_type": chunk.StructureNodeType,
					"canonical_path":      chunk.CanonicalPath,
					"item_index":          chunk.ItemIndex,
					"char_count":          chunk.CharCount,
					"token_count":         chunk.TokenCount,
					"embedding_model":     m.model,
					"create_time":         time.Now(),
					"update_time":         time.Now(),
					"status":              int8(0),
				},
			})
		}
	}
	return docs
}
