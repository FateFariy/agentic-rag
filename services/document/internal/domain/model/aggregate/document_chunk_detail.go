package aggregate

import (
	"github.com/swiftbit/know-agent/pkg/utils"
	"github.com/swiftbit/know-agent/services/document/internal/domain/model/entity"
)

type DocumentChunkDetail struct {
	DocumentId    int64
	TaskId        int64
	PlanId        int64
	Chunk         *entity.DocumentChunk
	ParentChunk   *entity.DocumentParentChunk
	SiblingChunks []*entity.DocumentChunk
}

func (d *DocumentChunkDetail) FillParentInfo(parentChunk *entity.DocumentParentChunk) {
	if parentChunk != nil {
		d.ParentChunk = parentChunk
		d.ParentChunk.FillEnumName()
		d.Chunk.FillParentInfo(parentChunk)
		d.Chunk.FillEnumName()
		utils.ForEach(d.SiblingChunks, func(index int, item *entity.DocumentChunk) {
			item.FillParentInfo(parentChunk)
			item.FillEnumName()
		})
	}
}
