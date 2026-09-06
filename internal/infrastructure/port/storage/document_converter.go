package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/indexer/milvus2"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus/client/v2/column"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

func DocumentConverter(vector *milvus2.VectorConfig, _ *milvus2.SparseVectorConfig) func(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]column.Column, error) {
	return func(ctx context.Context, docs []*schema.Document, vectors [][]float64) ([]column.Column, error) {
		ids := make([]string, 0, len(docs))
		contents := make([]string, 0, len(docs))
		vecs := make([][]float32, 0, len(docs))
		documentIds := make([]int64, 0, len(docs))
		taskIds := make([]int64, 0, len(docs))
		planIds := make([]int64, 0, len(docs))
		parentBlockIds := make([]int64, 0, len(docs))
		chunkNos := make([]int32, 0, len(docs))
		sourceTypes := make([]int32, 0, len(docs))
		sectionPaths := make([]string, 0, len(docs))
		structureNodeIds := make([]int64, 0, len(docs))
		structureNodeTypes := make([]int32, 0, len(docs))
		canonicalPaths := make([]string, 0, len(docs))
		itemIndices := make([]int32, 0, len(docs))
		charCounts := make([]int32, 0, len(docs))
		tokenCounts := make([]int32, 0, len(docs))
		embeddingModels := make([]string, 0, len(docs))
		createTimes := make([]time.Time, 0, len(docs))
		updateTimes := make([]time.Time, 0, len(docs))
		statuses := make([]int8, 0, len(docs))

		// Determine if we need to handle dense vectors
		denseVectorField := ""
		if vector != nil {
			denseVectorField = vector.VectorField
		}

		for idx, doc := range docs {
			ids = append(ids, doc.ID)
			contents = append(contents, doc.Content)
			documentIds = append(documentIds, utils.Convert[int64](doc.MetaData[vo.MetaDocumentId]))
			taskIds = append(taskIds, utils.Convert[int64](doc.MetaData[vo.MetaTaskId]))
			planIds = append(planIds, utils.Convert[int64](doc.MetaData[vo.MetaPlanId]))
			parentBlockIds = append(parentBlockIds, utils.Convert[int64](doc.MetaData[vo.MetaParentBlockId]))
			chunkNos = append(chunkNos, utils.Convert[int32](doc.MetaData[vo.MetaChunkNo]))
			sourceTypes = append(sourceTypes, utils.Convert[int32](doc.MetaData[vo.MetaSourceType]))
			sectionPaths = append(sectionPaths, utils.Convert[string](doc.MetaData[vo.MetaSectionPath]))
			structureNodeIds = append(structureNodeIds, utils.Convert[int64](doc.MetaData[vo.MetaStructureNodeId]))
			structureNodeTypes = append(structureNodeTypes, utils.Convert[int32](doc.MetaData[vo.MetaStructureNodeType]))
			canonicalPaths = append(canonicalPaths, utils.Convert[string](doc.MetaData[vo.MetaCanonicalPath]))
			itemIndices = append(itemIndices, utils.Convert[int32](doc.MetaData[vo.MetaItemIndex]))
			charCounts = append(charCounts, utils.Convert[int32](doc.MetaData["charCount"]))
			tokenCounts = append(tokenCounts, utils.Convert[int32](doc.MetaData["tokenCount"]))
			embeddingModels = append(embeddingModels, utils.Convert[string](doc.MetaData["embeddingModel"]))
			createTimes = append(createTimes, utils.Convert[time.Time](doc.MetaData["createTime"]))
			updateTimes = append(updateTimes, utils.Convert[time.Time](doc.MetaData["updateTime"]))
			statuses = append(statuses, utils.Convert[int8](doc.MetaData["status"]))

			var sourceVec []float64
			if len(vectors) == len(docs) {
				sourceVec = vectors[idx]
			} else {
				sourceVec = doc.DenseVector()
			}

			// Dense vector is required when vectorField is set (dense-only or hybrid mode).
			if denseVectorField != "" {
				if len(sourceVec) == 0 {
					return nil, fmt.Errorf("vector data missing for document %d (id: %s)", idx, doc.ID)
				}
				vec := make([]float32, len(sourceVec))
				for i, v := range sourceVec {
					vec[i] = float32(v)
				}
				vecs = append(vecs, vec)
			}

		}

		columns := []column.Column{
			column.NewColumnVarChar("id", ids),
			column.NewColumnVarChar("content", contents),
			column.NewColumnInt64("document_id", documentIds),
			column.NewColumnInt64("task_id", taskIds),
			column.NewColumnInt64("plan_id", planIds),
			column.NewColumnInt64("parent_block_id", parentBlockIds),
			column.NewColumnInt32("chunk_no", chunkNos),
			column.NewColumnInt32("source_type", sourceTypes),
			column.NewColumnVarChar("section_path", sectionPaths),
			column.NewColumnInt64("structure_node_id", structureNodeIds),
			column.NewColumnInt32("structure_node_type", structureNodeTypes),
			column.NewColumnVarChar("canonical_path", canonicalPaths),
			column.NewColumnInt32("item_index", itemIndices),
			column.NewColumnInt32("char_count", charCounts),
			column.NewColumnInt32("token_count", tokenCounts),
			column.NewColumnVarChar("embedding_model", embeddingModels),
			column.NewColumnTimestamptz("create_time", createTimes),
			column.NewColumnTimestamptz("update_time", updateTimes),
			column.NewColumnInt8("status", statuses),
		}

		if denseVectorField != "" {
			dim := 0
			if len(vecs) > 0 {
				dim = len(vecs[0])
			}
			columns = append(columns, column.NewColumnFloatVector(denseVectorField, dim, vecs))
		}

		return columns, nil
	}
}
