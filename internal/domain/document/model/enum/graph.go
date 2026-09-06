package enum

import "strings"

// GraphPersistenceOutcome 图谱持久化结果枚举
type GraphPersistenceOutcome = string

const (
	GraphPersistenceOutcomeSuccess  GraphPersistenceOutcome = "SUCCESS"
	GraphPersistenceOutcomeFailed   GraphPersistenceOutcome = "FAILED"
	GraphPersistenceOutcomeEmpty    GraphPersistenceOutcome = "EMPTY"
	GraphPersistenceOutcomeDegraded GraphPersistenceOutcome = "DEGRADED"
)

// GraphRagEntityTypes 允许抽取的实体类型（通用受控列表，模板注入用）
// 面向通用知识库语料（法规/技术/业务/文档皆可），避免开放域造成实体类型爆炸
var GraphRagEntityTypes = map[string]struct{}{
	"组织": {}, "人物": {}, "地点": {}, "产品": {}, "概念": {},
	"事件": {}, "时间": {}, "规则": {}, "技术": {}, "标准": {}, "项目": {}, "类别": {},
}

// GraphRagRelationFallback 关系类型兜底值：白名单外一律落该关系类型，保留 predicate_quote 原文谓词
const GraphRagRelationFallback = "相关于"

// GraphRagRelationWhitelist 受控关系类型白名单（通用语义集合，命中→动态关系类型；未命中→相关于 兜底）
// 覆盖分类/组成/定义/引用/依据/约束/变更等常见本体语义，语料具体细节由 predicate_quote 原文谓词承载
var GraphRagRelationWhitelist = map[string]struct{}{
	"属于": {}, "包含": {}, "定义": {}, "引用": {}, "适用于": {},
	"依据": {}, "要求": {}, "禁止": {}, "取代": {},
}

// NormalizeRelationType 归一化关系类型：白名单内原样返回，否则落 相关于 兜底
func NormalizeRelationType(rt string) string {
	if rt != "" {
		if _, ok := GraphRagRelationWhitelist[rt]; ok {
			return rt
		}
	}
	return GraphRagRelationFallback
}

// NormalizeEntityType 实体类型受控归一：白名单内原样返回，未命中归入"概念"防类型爆炸
func NormalizeEntityType(t string) string {
	if t = strings.TrimSpace(t); t != "" {
		if _, ok := GraphRagEntityTypes[t]; ok {
			return t
		}
	}
	return "概念"
}
