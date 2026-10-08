package enum

// QueryType 查询类型枚举
type QueryType = string

const (
	QueryTypeDocumentQA          QueryType = "document_qa"          // 文档问答: 基于知识库内容的问答
	QueryTypeFollowUp            QueryType = "follow_up"            // 追问: 基于上下文的连续对话
	QueryTypeStructureNavigation QueryType = "structure_navigation" // 结构导航: 文档结构/目录导航
	QueryTypeGraphRelation       QueryType = "graph_relation"       // 图谱关系: 知识库图谱关系查询
	QueryTypeGlobalSummary       QueryType = "global_summary"       // 全局摘要: 跨文档/全局内容总结
)
