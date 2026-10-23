package config

// ChunkConf 文档分块配置（knowledge 自带副本，用于索引切分参数）
type ChunkConf struct {
	RecursiveMaxChars           int     `json:",default=800"`  // 递归分块的最大字符数，默认800
	RecursiveOverlapChars       int     `json:",default=120"`  // 递归分块的重叠字符数，默认120
	SemanticMaxChars            int     `json:",default=700"`  // 语义分块的最大字符数，默认700
	SemanticMinChars            int     `json:",default=240"`  // 语义分块的最小字符数，默认240
	ParentChunkMaxChars         int     `json:",default=2200"` // 父块的最大字符数，默认2200
	ParentChunkOverlapChars     int     `json:",default=180"`  // 父块的重叠字符数，默认180
	ParentSemanticMaxChars      int     `json:",default=1600"` // 父块的语义最大字符数，默认1600
	ParentSemanticMinChars      int     `json:",default=480"`  // 父块的语义最小字符数，默认480
	SemanticSimilarityThreshold float64 `json:",default=0.18"` // 语义相似度阈值，默认0.18
}
