package vo

// QuestionRewriteResult 问题改写结果
type QuestionRewriteResult struct {
	RewrittenQuestion string // 改写后的问题
	RawModelOutput    string // 原始模型输出
}
