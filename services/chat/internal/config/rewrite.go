package config

// RewriteConf 问题改写配置
type RewriteConf struct {
	Enabled     bool    `json:",default=true"`  // 是否启用问题改写
	Temperature float32 `json:",default=0.1"`   // 温度参数
	TopP        float32 `json:",default=0.3"`   // TopP参数
	Thinking    bool    `json:",default=false"` // 是否启用思考过程
}
