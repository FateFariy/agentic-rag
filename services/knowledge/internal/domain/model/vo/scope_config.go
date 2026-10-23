package vo

import "fmt"

// ScopeConfig 检索范围收窄相关阈值配置
type ScopeConfig struct {
	MinIntentScore      float64 // 节点计入命中的次级阈值（意图分数下限）
	ConfidenceThreshold float64 // topScore 达到该阈值则判定为 DIRECTED（定向检索）
	SupplementRatio     float64 // 补充路比例，取值区间 [0,1]
}

// NewDefaultScopeConfig 返回检索范围收窄阈值的默认值
func NewDefaultScopeConfig() *ScopeConfig {
	return &ScopeConfig{
		MinIntentScore:      0.4,
		ConfidenceThreshold: 0.6,
		SupplementRatio:     0.25,
	}
}

// Clone 深拷贝一份 ScopeConfig，nil 接收者返回 nil
func (c *ScopeConfig) Clone() *ScopeConfig {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

// Validate 校验阈值合法性
func (c *ScopeConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("ScopeConfig 不能为 nil")
	}
	if c.MinIntentScore < 0 {
		return fmt.Errorf("ScopeConfig.MinIntentScore 必须 >= 0，实际值: %v", c.MinIntentScore)
	}
	if c.ConfidenceThreshold <= 0 {
		return fmt.Errorf("ScopeConfig.ConfidenceThreshold 必须 > 0，实际值: %v", c.ConfidenceThreshold)
	}
	if c.MinIntentScore >= c.ConfidenceThreshold {
		return fmt.Errorf("ScopeConfig.MinIntentScore 必须小于 ConfidenceThreshold，实际值: MinIntentScore=%v, ConfidenceThreshold=%v",
			c.MinIntentScore, c.ConfidenceThreshold)
	}
	if c.SupplementRatio < 0 || c.SupplementRatio > 1 {
		return fmt.Errorf("ScopeConfig.SupplementRatio 必须在 [0,1] 区间内，实际值: %v", c.SupplementRatio)
	}
	return nil
}
