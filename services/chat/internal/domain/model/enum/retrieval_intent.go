package enum

type RetrievalIntent = string

const (
	RetrievalIntentGeneral   RetrievalIntent = "general"
	RetrievalIntentGraphRAG  RetrievalIntent = "graph_rag"
	RetrievalIntentStructure RetrievalIntent = "structure"
)
