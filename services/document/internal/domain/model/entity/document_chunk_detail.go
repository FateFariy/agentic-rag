package entity

import (
	"github.com/swiftbit/know-agent/pkg/utils"
)

type DocumentChunkDetail struct {
	DocumentId    int64
	TaskId        int64
	PlanId        int64
	Chunk         *DocumentChunk
	ParentChunk   *DocumentParentChunk
	SiblingChunks []*DocumentChunk
}

func (d *DocumentChunkDetail) FillParentInfo(parentChunk *DocumentParentChunk) {
	if parentChunk != nil {
		d.ParentChunk = parentChunk
		d.ParentChunk.FillEnumName()
		d.Chunk.FillParentInfo(parentChunk)
		d.Chunk.FillEnumName()
		utils.ForEach(d.SiblingChunks, func(index int, item *DocumentChunk) {
			item.FillParentInfo(parentChunk)
			item.FillEnumName()
		})
	}
}
