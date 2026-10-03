package enum

// RouteStatus 知识路由状态
type RouteStatus = string

const (
	RouteStatusSuccess       RouteStatus = "SUCCESS"        // 路由成功
	RouteStatusLowConfidence RouteStatus = "LOW_CONFIDENCE" // 置信度不足，已降级扩范围
	RouteStatusFailed        RouteStatus = "FAILED"         // 路由失败
)
