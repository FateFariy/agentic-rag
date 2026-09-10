package enum

type RetrievalIntent = string

const (
	RetrievalIntentGeneral   RetrievalIntent = "general"
	RetrievalIntentTable     RetrievalIntent = "table"
	RetrievalIntentGraphRAG  RetrievalIntent = "graph_rag"
	RetrievalIntentRaptor    RetrievalIntent = "raptor"
	RetrievalIntentStructure RetrievalIntent = "structure"
)
