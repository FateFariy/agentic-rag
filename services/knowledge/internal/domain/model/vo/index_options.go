package vo

type IndexingOptions struct {
	Chunk    ChunkOptions         `json:"chunk"`
	GraphRag GraphRagBuildOptions `json:"graphRag"`
}

type ChunkOptions struct {
	ChildRecursiveMaxChars           int     `json:"childRecursiveMaxChars"`
	ChildRecursiveOverlapChars       int     `json:"childRecursiveOverlapChars"`
	ChildSemanticMaxChars            int     `json:"childSemanticMaxChars"`
	ChildSemanticMinChars            int     `json:"childSemanticMinChars"`
	ChildSemanticSimilarityThreshold float64 `json:"childSemanticSimilarityThreshold"`
	ParentChunkMaxChars              int     `json:"parentChunkMaxChars"`
	ParentChunkOverlapChars          int     `json:"parentChunkOverlapChars"`
	ParentSemanticMaxChars           int     `json:"parentSemanticMaxChars"`
	ParentSemanticMinChars           int     `json:"parentSemanticMinChars"`
}

type GraphRagBuildOptions struct {
	GraphRagBuildEnabled bool `json:"graphRagBuildEnabled"`
}
