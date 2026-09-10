# Know-RAG · 企业级知识库 Agentic RAG 系统

> 一套面向企业私有知识场景、可生产落地的 **Agentic RAG** 系统。以 **eino deep agent** 为决策核心，提供文档解析、结构化分块、向量 / 关键词 / 知识图谱（GraphRAG）混合召回、语义缓存、查询改写、会话记忆压缩、引用溯源、在线回答评估与全链路追踪等完整能力。

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Vue](https://img.shields.io/badge/Vue-3.5-42B883?logo=vuedotjs&logoColor=white)](https://vuejs.org)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](#许可证)
[![Framework](https://img.shields.io/badge/go--zero-1.10-orange)](https://github.com/zeromicro/go-zero)
[![Eino](https://img.shields.io/badge/Eino-0.9-FF6F00)](https://github.com/cloudwego/eino)
[![Milvus](https://img.shields.io/badge/Milvus-2.6-00A1B7)](https://milvus.io)
[![Ollama](https://img.shields.io/badge/Ollama-Embedding-000000)](https://ollama.com)
[![RocketMQ](https://img.shields.io/badge/RocketMQ-5.x-D40000)](https://rocketmq.apache.org)
[![MinIO](https://img.shields.io/badge/MinIO-Storage-C72E29)](https://min.io)

---

## 目录

- [项目概述](#项目概述)
- [项目亮点](#项目亮点)
- [核心功能](#核心功能)
- [技术架构](#技术架构)
- [项目结构](#项目结构)
- [技术栈](#技术栈)
- [核心流程](#核心流程)
- [数据库与基础设施](#数据库与基础设施)
- [可观测与追踪](#可观测与追踪)
- [接口概览](#接口概览)
- [快速开始](#快速开始)
- [设计文档](#设计文档)

---

## 项目概述

**Know-RAG** 是一套面向 **企业私有知识库** 场景的 **Agentic RAG** 服务，专注于解决以下工程问题：

- **结构化文档解析**：识别标题层级、章节、列表、表格、引用、附录等结构信号，构造文档骨架并自动生成分块策略。
- **高质量分块**：内置 `Structure/ Recursive / Semantic / LLM` 三种分块策略，按文档特征推荐，支持人工确认与微调。
- **Agentic 检索决策**：基于 agent 自主决定是否检索、检索几次、是否需要回看文档结构，`Think → Tool → Answer` 多轮循环后给出带引用的答案。
- **混合检索与重排**：向量召回 + 关键词召回 + 图谱（GraphRAG）召回 + RRF 融合 + Rerank，统一召回质量。
- **语义缓存**：对「改写后的问题」做向量召回 + 分区间命中判定（高置信直击 / 灰区大模型终判），命中后复用历史检索产物甚至答案草稿，显著降低延迟与成本。
- **会话级记忆压缩**：`summary_compression` 策略自动生成结构化长期记忆，兼顾长上下文与成本。
- **在线回答评估**：内置 Faithfulness / Relevancy / Context Precision 三类评估器，在每轮对话中并发打分并落库。
- **引用溯源与可观测**：每一次会话均产出 `DebugTrace` + `StageTrace` + `ChannelExecution` + `RetrievalResult`，可逐阶段回放。
- **异步流水线**：基于 RocketMQ 的「解析 → 索引构建 → 语义缓存写入」解耦，主链路不受长任务阻塞。
- **知识图谱（GraphRAG）**：文档入库时抽取实体 / 关系并构建实体图谱（Neo4j），结合 PageRank 重要性排序与 Louvain 社区发现生成社区摘要；检索时作为增强通道（种子实体 + n-hop 路径扩展 + 社区召回 + 证据回流），与向量 / 关键词通道并行融合。

系统采用 **go-zero 微服务框架** + **领域驱动设计（DDD）分层**架构，模块边界清晰、易于二次开发。

---

## 项目亮点

1. **Agentic 架构**：对话主链路以 `eino adk` 的 agent 取代固定编排，模型自主规划检索、调用工具、收敛答案，并支持思考过程流式透出。
2. **DDD + 手动注入的清晰架构**：领域 / 基础设施 / 触发器严格分层，依赖在 `cmd/bootstrap.go` 中按域手动组装，二次开发不污染核心。
3. **工具化检索能力**：`knowledge_base_search`（混合检索）与 `route_document_structure`（文档结构导航）以 eino 工具形式注册，新增工具只需实现领域 `Tool` 接口。
4. **多策略分块自适应**：`Recursive / Semantic / LLM` 分块按文档特征自动推荐，支持人工确认与重跑索引。
5. **语义缓存双复用策略**：`ReuseRetrievalOnly`（复用检索、重跑生成）与 `ReuseAnswerAndRetrieval`（检索与答案一起复用），命中判定带降级兜底，绝不误伤正确性。
6. **混合检索 + RRF + Rerank**：向量 + 关键词双通道并行，RRF 融合、父块回溯、Rerank 精排，阈值与权重全部可配。
7. **完整可观测**：每轮对话产出 `DebugTrace` + `StageTrace` + `ChannelExecution` + `RetrievalResult` + 评估明细，可逐阶段回放。
8. **可插拔基础设施**：Milvus / MinIO / MQ / LLM / Embedding / Rerank 全部在 `internal/infrastructure/port/` 内替换实现，迁移成本低。
9. **知识图谱（GraphRAG）**：实体抽取 + PageRank 重要性排序 + Louvain 社区发现 + LLM 社区摘要构建 Neo4j 实体图谱；检索侧以「增强通道」形式与向量 / 关键词并行，未装配 / 未请求 / 失败均自动降级，不影响主链路。

---

## 核心功能

### 1. 文档管理
- 文档上传、解析、分页查询、删除
- 结构化骨架识别（章 / 节 / 小节 / 列表 / 表格 / 代码块 / 引用 / 附录）
- 文档画像（`document_profile`）自动生成与重建
- 分块策略推荐与人工确认（`document_strategy_plan` / `_step`）
- 向量 + 关键词索引构建、任务日志与重试

### 2. 知识管理
- 知识库（`knowledge_base`）与知识范围（Scope）树形管理
- 知识主题（Topic）与文档的多对多关联
- 文档画像与主题路由（`knowledge_document_profile`）
- 范围 / 主题 / 文档三级排序器（`rank`）+ 路由追踪（`knowledge_route_trace`）

### 3. 智能问答
- 流式聊天（SSE）：思考过程、工具调用、答案、引用、推荐问题分事件推送
- 多模式：`document`（指定文档）/ `auto_document`（自动路由到主文档）
- 查询改写（指代消解 + 上下文补全）
- 混合检索（向量 + 关键词 + 图谱 GraphRAG + RRF + Rerank）
- 知识图谱检索（GraphRAG）：实体图谱增强召回（依赖 Neo4j，可选）
- 会话级长期记忆 + 上下文压缩
- 实时引用 + 章节级溯源
- 语义缓存（`SemanticCache`）
- 回答质量评估（Faithfulness / Relevancy / Context Precision）
- 追问推荐（`Recommendation`）
- 会话停止 / 重置 / 摘要重建

### 4. Agentic 执行（deep agent）
`AgentStage` 通过 `AgentRunner` 调用 `eino adk` 的 deep agent（`prebuilt/deep`），在一次会话内自主完成：

```
系统指令（含记忆 / 日期 / 知识库范围，由中间件注入）
      │
      ▼
 ┌───────────────────────────────────────────┐
 │  Think → 决策：是否检索 / 检索什么 / 是否 │
 │  需要回看文档结构 / 证据是否足够作答      │
 └───────┬───────────────────────┬───────────┘
         │ 需要证据              │ 证据充分
         ▼                       ▼
   调用工具（可多轮）       生成最终答案（带 [1][2] 引用）
```

- 工具：`knowledge_base_search`（知识库混合检索）、`route_document_structure`（文档章节 / 条目导航）
- 预算约束：`MaxModelCallsPerRun` / `MaxModelCallsPerThread` / `MaxToolCallsPerRun` / `MaxToolCallsPerThread`
- 中间件：`context_inject_middleware` 在 `BeforeAgent` 注入领域上下文（记忆摘要、当前日期、知识库范围等）

### 5. 语义缓存
命中判定采用分区间策略（`SemanticCacheStage`）：

| 区间 | 条件 | 行为 |
| --- | --- | --- |
| 未召回 | 候选为空或相似度 < `SimilarityThreshold` | 未命中 |
| 高置信区 | 最高分 ≥ `DirectHitThreshold` | 直接命中 Top1，零大模型开销 |
| 灰色区 | 介于两者之间 | `JudgeEnabled` 时大模型批量等价终判（置信度需 ≥ `ConfidenceFloor`），否则保守未命中 |

- 判定基于 **改写后的问题** 向量，命中即语义真正等价
- 命中后回填检索产物：`Route`、`AnswerEvaluate` 阶段跳过；`ReuseAnswerAndRetrieval` 策略下 `Agent` 直接发布缓存答案草稿，`ReuseRetrievalOnly` 策略下仍运行 agent 基于复用证据重新生成
- 两种复用策略：`ReuseRetrievalOnly`（重跑生成）/ `ReuseAnswerAndRetrieval`（直接发布缓存答案草稿）
- 写入走 RocketMQ 异步落库（`CacheWriteStage` → `know-agent-semantic-cache-write`），不阻塞主链路
- 实时性敏感（`RequiresRealTimeSearch` / `RequiresCurrentDateAnchoring`）的查询自动跳过缓存

### 6. 记忆与上下文
- `summary_compression`：定时压缩历史，生成结构化 `SummaryPayload`（目标 / 事实 / 偏好 / 已解决 / 待办 / 检索提示）
- `sliding_window`：滑动窗口策略
- 自动覆盖轮次追踪、版本管理、摘要重建

### 7. 回答评估
- 在线（每轮对话）：`Faithfulness` / `Relevancy` / `Context Precision`，各评估器独立 goroutine 并发产出

### 8. 知识图谱（GraphRAG）
文档入库时自动构建实体图谱，检索时作为与向量 / 关键词并列的增强通道（图谱为「增强依赖」，**Neo4j 不可用时自动降级，不阻断其余通道**）。

**构建（文档索引流水线）**：`graphRagStage` 在索引阶段追加执行 `Builder.RebuildDocumentGraph`，依次：
1. **LLM 抽取**（`extract`）：受控实体类型（组织 / 人物 / 地点 / 产品 / 概念 / 事件 / …）与受控关系白名单（属于 / 包含 / 定义 / 引用 / 依据 / 要求 / 禁止 / 取代 …），白名单外关系落 `相关于` 兜底，谓词原文保留在 `predicate_quote`；
2. **实体消解合并**（`resolver`）：按归一化名合并别名、累加 `mention_count`；
3. **PageRank 重要性排序**（`pagerank`）+ 成本闸门（超 `MaxEntities` / `MaxRelations` 按重要性截断）；
4. **社区发现**（`community`）：并查集连通分量 + 可选 Louvain 细化（`LouvainEnabled`），产出 `:Community` 节点与 `IN_COMMUNITY` 边；
5. **LLM 社区摘要**（`llm_community_reporter`，M3）：仅前 `CommunityReportTopN` 个重要社区生成标题 / 摘要；
6. **事务化落库 Neo4j**（按 `document_id + task_id` 全删全插，幂等重建）；
7. **图卡片投影**（`typed_card`，M3）：实体 / 关系 / 社区三类图卡片写入向量库，增强语义召回。

构建失败不影响文档索引主任务（构建结果状态 `SUCCESS / FAILED / EMPTY / DEGRADED` 由图谱阶段记录）。

**检索（对话检索管线）**：`GraphRAGRetrievalChannel`（M1）与向量 / 关键词通道并行：
- **触发**：Agent 声明 `graph_rag` 检索意图，或问题类型识别为 `graph_relation`；
- **策略**：种子实体命中 → n-hop 变长路径扩展（硬限 `SearchMaxHops`=2）→ 社区召回 → 证据回流为 `DocumentChunk`；
- **融合**：经 RRF + Rerank 与向量 / 关键词结果统一精排。

---

## 技术架构

### 整体架构图

```
                ┌────────────────────────────────────────┐
                │            HTTP / SSE Client           │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │   go-zero REST API Layer (api/*)       │
                │  - /chat   - /manage/document          │
                │  - /manage/knowledge                   │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │  Trigger Handler（HTTP / MQ 入口）      │
                │  - chat_service   - document_service   │
                │  - knowledge_service                   │
                └──────────────────┬─────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────┐
                │       Domain Logic (DDD)               │
                │  ┌────────────┬────────────┬──────────┐│
                │  │  Chat      │ Document   │ Knowledge││
                │  │  • 改写     │ • 解析     │ • 路由    ││
                │  │  • 缓存     │ • 分块     │ • 主题    ││
                │  │  • Agent    │ • 索引     │ • 画像    ││
                │  │  • 记忆     │ • 策略     │           ││
                │  │  • 评估     │           │           ││
                │  │  • 推荐     │           │           ││
                │  └──────┬─────┴────────────┴──────────┘│
                │         │ 检索引擎（Pipeline）          │
                └─────────┼───────────────────────────────┘
                          │
        ┌─────────────────▼─────────────────┐
        │  Eino Agent 层                    │
        │  deep agent(adk) + Tools          │
        │  + 领域中间件 + CheckPoint        │
        └─────────────────┬─────────────────┘
                          │
        ┌─────────────────▼─────────────────────────────┐
        │  Infrastructure Adapters                      │
        │  MySQL · Redis · MinIO · Milvus · RocketMQ    │
        │  Neo4j(可选) · LLM(Ark/Ollama) · Embedding    │
        │  (Ollama) · Rerank                            │
        └───────────────────────────────────────────────┘

前端（know-agent/front）
                ┌────────────────────────────────────────┐
                │  Vue 3 + Vite + Element Plus + Pinia   │
                │  - ChatPage（用户对话）                │
                │  - AdminLayout / Dashboard             │
                │  - DocumentList / DocumentDetail       │
                │  - KnowledgeRoute / RouteTrace         │
                │  - ObservabilityList / Session / Detail│
                └────────────────────────────────────────┘
```

### 模块分层（DDD）
- `api/`：接口定义（`.api` 文件）+ HTTP 路由（go-zero 生成）
- `cmd/`：启动入口与依赖组装
- `internal/config`：配置加载
- `internal/domain/<aggregate>/`
  - `adapter/`：外部接口契约（port / repository / gateway）
  - `logic/`：业务实现
  - `model/entity|vo|enum`：领域实体、值对象、枚举
- `internal/infrastructure/`：基础设施实现（MySQL、Milvus、MinIO、Redis、RocketMQ、LLM、Embedding、Rerank、缓存等）
- `internal/trigger/`：消费者 / 处理器适配入口
- `internal/server/`：服务组装
- `internal/svc/`：服务上下文
- `common/`：通用工具（`logx`、`utils`：Snowflake、JSON、字符串、切片、三元等）

### 依赖手动注入
`cmd/bootstrap.go` 在启动期按域（chat / document / knowledge / infrastructure / server）手动组装完整服务图，无需代码生成，依赖关系直观、易于调试。

### 对话链路（Conversation Chain）
由 `conversation.NewChain` 装配，按 `Stage.Order()` 排序后顺序执行的 10 阶段流水线：

```
Start → MemoryLoad → QueryRewrite → SemanticCache
     → Route → Agent → AnswerEvaluate → Recommend
     → CacheWrite → End
```

| 阶段 | 顺序 | 说明 | 条件 |
| --- | --- | --- | --- |
| `Start` | 0 | 分布式锁抢占 + 会话初始化 | 否 |
| `MemoryLoad` | 10 | 历史摘要 / 近期转录 / 上下文装载 | 否 |
| `QueryRewrite` | 20 | 指代消解、歧义消除、子问题拆分 | 执行计划就绪 |
| `SemanticCache` | 30 | 向量召回 + 分区间命中判定 | 启用且非实时 / 日期敏感 |
| `Route` | 40 | 知识范围解析、知识路由、主文档选择 | 执行计划就绪且未命中缓存 |
| `Agent` | 50 | deep agent 自主检索与生成 | 否 |
| `AnswerEvaluate` | 60 | Faithfulness / Relevancy / Context Precision 并发打分 | 有检索结果且未命中缓存 |
| `Recommend` | 70 | 追问推荐 | 否 |
| `CacheWrite` | 80 | 语义缓存异步写入（RocketMQ） | 非「命中且复用答案」 |
| `End` | 90 | 引用绑定、落库收尾（TraceRecorder） | 否 |

失败与中断统一收敛：`ErrSessionRunning` 回写「执行中」提示，`context.Canceled` 走会话停止流程，其余错误按失败态落库。

### 检索管线（Retrieval Pipeline）
```
ChannelRetrieval → Fusion(RRF) → ParentElevation → Rerank
                → FinalTopK → TopDocumentSummary → Observation
```
- `ChannelRetrieval`：并行执行向量 / 关键词 / 图谱（GraphRAG）通道，按子问题并发
- `Fusion`：RRF 倒数排名融合
- `ParentElevation`：回溯到父块，补足上下文
- `Rerank`：可选精排（DashScope 兼容接口）
- `FinalTopK`：按 TopK 截断
- `TopDocumentSummary`：顶级文档摘要补充全局视角
- `Observation`：落库通道执行 / 检索结果明细
- `GraphRAG` 通道（增强）：种子实体 + n-hop 路径扩展 + 社区召回 + 证据回流；未装配 / 未请求 / 失败均降级为空，不阻断其余通道

### 异步处理
- `trigger/consumer/parse_document.go`：消费解析任务 → 抽取结构 → 写 MinIO → 落库
- `trigger/consumer/build_index.go`：消费索引任务 → 选分块策略 → Embedding → 写 Milvus
- `trigger/consumer/semantic_cache_write.go`：消费语义缓存写入任务 → 落 `chat_cache_entry`
- `internal/infrastructure/port/mq`：RocketMQ 生产者封装与重试策略

---

## 项目结构

```
know-agent/
├── api/                              # 接口层（go-zero .api 定义）
│   ├── chat/                         #   - /chat（SSE 对话、会话与观测查询）
│   ├── document/                     #   - /manage/document
│   └── knowledge/                    #   - /manage/knowledge
├── cmd/                              # 启动入口
│   ├── main.go                       #   - 加载配置、启动 Server
│   └── bootstrap.go                  #   - 手动依赖组装（Chain / Agent / 检索引擎 / 评估器）
├── common/                           # 通用工具
│   ├── biz_error.go                  #   - 业务错误
│   ├── model.go                      #   - 通用模型
│   ├── option.go                     #   - 通用选项
│   ├── response.go                   #   - 统一响应
│   ├── logx/                         #   - 日志
│   └── utils/                        #   - 工具集（Snowflake、字符串、切片、三元等）
├── doc/                              # OpenAPI 文档
│   ├── chat.json
│   ├── document.json
│   └── knowledge.json
├── etc/                              # 配置文件
│   ├── config-dev.yaml               #   - 开发环境配置
│   ├── config-prod.yaml              #   - 生产环境配置
│   ├── document_chunk_collection.json#   - Milvus 文档块集合结构
│   ├── semantic_cache_collection.json#   - Milvus 语义缓存集合结构
│   └── schema.sql                    #   - MySQL DDL
├── test/                             # 集成测试（依赖真实基础设施）
├── internal/                         # 内部实现
│   ├── config/                       #   - 配置结构
│   ├── convert/                      #   - 类型转换器
│   ├── domain/                       #   - 领域层
│   │   ├── chat/                     #     - 聊天域
│   │   │   ├── logic/conversation/   #       - 10 阶段 Chain + AgentStage + 缓存阶段
│   │   │   │   ├── middleware/       #         - 领域层 Agent 中间件
│   │   │   │   └── tool/             #         - 知识库检索 / 文档结构导航工具
│   │   │   ├── logic/retrieval/      #       - 检索管线（channel/fuse/7 个阶段）
│   │   │   ├── logic/evaluate/       #       - 4 类评估器
│   │   │   ├── logic/rewrite/        #       - 查询改写
│   │   │   ├── logic/memory/         #       - 会话记忆 + 摘要压缩
│   │   │   ├── logic/route/          #       - 文档结构路由
│   │   │   └── logic/recommend/      #       - 追问推荐
│   │   ├── document/                 #     - 文档域
│   │   │   └── logic/process/        #       - 文档处理流水线
│   │   │       ├── parse/            #         - 解析
│   │   │       ├── analysis/         #         - 分析（Chain + Stage）
│   │   │       ├── chunk/            #         - 分块（recursive / semantic / llm）
│   │   │       └── index/            #         - 索引构建（含 GraphRAG 图谱构建 graph/）
│   │   ├── knowledge/                #     - 知识域（Scope / Topic / Router / Rank）
│   │   └── callbacks/                #     - 回调域（编排 / 追踪回调）
│   ├── error/                        #   - 错误码
│   ├── infrastructure/               #   - 基础设施适配
│   │   ├── model/                    #     - ORM 模型
│   │   ├── persistence/              #     - MySQL 仓储
│   │   ├── observability/            #     - 追踪与指标注册
│   │   └── port/                     #     - 外部端口
│   │       ├── agent/                #       - eino deep agent + 工具 / 中间件适配器
│   │       ├── vector/               #       - Milvus 向量
│   │       ├── keyword/              #       - Milvus 关键词（全文）
│   │       ├── storage/              #       - MinIO + ElasticSearch + Neo4j 图谱存储
│   │       ├── mq/                   #       - RocketMQ 生产者 / 消费者
│   │       ├── llm/                  #       - Ark / Ollama Chat Model
│   │       ├── emb/                  #       - Ollama Embedder
│   │       ├── rerank/               #       - DashScope Rerank
│   │       ├── cache/                #       - 语义缓存存储
│   │       ├── lock/                 #       - Redis 分布式锁
│   │       ├── parser/               #       - Markdown / 文档解析
│   │       ├── gateway/              #       - 域间适配网关
│   │       ├── prompt/               #       - 提示词模板渲染
│   │       ├── tokenize/             #       - GSE 分词
│   │       └── config/               #       - 本地配置
│   ├── server/                       #   - HTTP 服务
│   ├── svc/                          #   - 服务上下文
│   └── trigger/                      #   - 触发器
│       ├── consumer/                 #     - MQ 消费者（parse / build_index / 缓存写入）
│       └── handler/                  #     - 业务处理器
├── front/                            # 前端工程（Vue 3 + Vite）
│   ├── src/
│   │   ├── views/
│   │   │   ├── ChatPage.vue          #   - 用户对话主页
│   │   │   └── admin/                #   - 管理后台
│   │   │       ├── AdminLayoutView.vue
│   │   │       ├── AdminDashboardView.vue
│   │   │       ├── AdminDocumentListView.vue
│   │   │       ├── AdminDocumentDetailView.vue
│   │   │       ├── AdminKnowledgeRouteView.vue
│   │   │       ├── AdminKnowledgeRouteTraceView.vue
│   │   │       ├── AdminObservabilityListView.vue
│   │   │       ├── AdminObservabilitySessionView.vue
│   │   │       └── AdminObservabilityDetailView.vue
│   │   ├── components/               #   - chat / admin / common 组件
│   │   └── api/                      #   - 后端接口封装
│   └── package.json                  #   - Vue 3.5 / Vite 6 / Element Plus / Pinia
├── Dockerfile                        # 多阶段构建镜像
├── docker-compose.yml                # 前后端容器编排（frontend + backend）
├── go.mod / go.sum                   # 依赖
└── README.md                         # 本文档
```

---

## 技术栈

| 类别 | 选型 | 用途 |
| --- | --- | --- |
| 语言（后端） | Go 1.25 | 主语言 |
| HTTP 框架 | [go-zero](https://github.com/zeromicro/go-zero) 1.10 | RESTful API / 路由 / 配置 |
| Agent 编排 | [Cloudwego Eino](https://github.com/cloudwego/eino) 0.9（adk / prebuilt-deep） | deep agent、工具、中间件 |
| LLM 模型 | 火山方舟 Ark（豆包系列）/ Ollama | Chat、改写、推荐、判定 |
| 向量模型 | Ollama（默认 `qwen3-embedding:4b`，1024 维） | Embedding、语义缓存向量化 |
| 重排 | DashScope 兼容接口（默认 `qwen3-rerank`） | Rerank 精排 |
| 向量数据库 | Milvus 2.6 | 向量 + 全文（关键词）检索 |
| 关系数据库 | MySQL（GORM） | 业务元数据、对话、追踪、评估 |
| 缓存 / 锁 | Redis（go-redis + redsync） | 分布式锁、会话并发控制 |
| 对象存储 | MinIO | 原始文档、解析文本、解析产物 |
| 消息队列 | Apache RocketMQ（client-go v2） | 异步解析 / 索引 / 缓存写入 |
| 图数据库 | Neo4j（neo4j-go-driver/v5，可选） | 知识图谱存储与图谱检索（GraphRAG） |
| 检索辅助 | ElasticSearch（gse 分词） | 知识路由侧词法召回 |
| 配置 | YAML（go-zero conf） | 多环境配置 |
| 语言（前端） | TypeScript 5.9 + Vue 3.5 | 单页应用 |
| 前端构建 | Vite 6 | 开发与构建 |
| 前端 UI | Element Plus 2.14 | 组件库 |
| 前端状态 | Pinia 2.3 + persistedstate | 状态管理 |
| 容器化 | Docker / docker-compose | 镜像构建与编排 |

---

## 核心流程

### 单轮对话主流程
```
用户问题
  │
  ▼
[1] Start ── 分布式锁抢占 + 会话初始化
  │
  ▼
[2] MemoryLoad ── 历史摘要 / 近期转录 / 知识库选择快照
  │
  ▼
[3] QueryRewrite ── 指代消解 + 子问题拆分
  │
  ▼
[4] SemanticCache ── 向量召回 → 高置信直击 / 灰区大模型终判
  │        命中 → 回填检索产物，跳过 Route 与评估
  ▼
[5] Route ── 知识范围解析 → 知识路由 → 主文档选择（auto_document）
  │
  ▼
[6] Agent ── deep agent：Think → knowledge_base_search /
  │          route_document_structure → 证据充分后生成带引用答案
  ▼
[7] AnswerEvaluate（条件）── Faithfulness / Relevancy / Context Precision 并发打分
  │
  ▼
[8] Recommend ── 追问推荐
  │
  ▼
[9] CacheWrite（条件）── 语义缓存条目经 RocketMQ 异步落库
  │
  ▼
[10] End ── 引用绑定 / 阶段追踪落库（TraceRecorder）
```

### 知识库范围与文档路由
- `document` 模式：记录影子路由（仅用于离线分析，失败只告警）
- `auto_document` 模式：解析知识库选择快照 → 双路输入（原始问题 + 改写问题）执行路由 → 按允许范围过滤候选 → 置信度达标（`AutoRoute.RecommendationThreshold`）则确定主文档
- 路由状态、置信度、候选数、是否降级等信息全部写入阶段快照，可在可观测页面回放

---

## 数据库与基础设施

### MySQL 关键表（`etc/schema.sql`）

| 分组 | 表 |
| --- | --- |
| 对话 | `chat_dialogue`、`chat_exchange`、`chat_exchange_trace_stage`、`chat_memory_summary` |
| 检索观测 | `chat_channel_execution`、`chat_retrieval_result` |
| 评估与缓存 | `chat_exchange_eval`、`chat_cache_entry` |
| 文档 | `document`、`document_block`、`document_chunk`、`document_parent_chunk`、`document_profile`、`document_strategy_plan`、`document_strategy_step`、`document_structure_node`、`document_task`、`document_task_log` |
| 知识 | `knowledge_base`、`knowledge_scope_node`、`knowledge_topic_node`、`knowledge_topic_document_relation`、`knowledge_document_profile`、`knowledge_route_trace` |

### Milvus（`etc/*_collection.json`）
- 文档块集合 `document_chunk`（`etc/document_chunk_collection.json`）
  - 主键 `id`；核心字段 `document_id / task_id / plan_id / parent_block_id / chunk_no / source_type / section_path / structure_node_id / structure_node_type / canonical_path / item_index / char_count / token_count / embedding_model / content`
  - 稠密向量 `vector`（1024 维，HNSW + `COSINE`）+ 稀疏向量 `sparse_vector`（BM25 函数由 `content` 自动生成，中文分词）
- 语义缓存集合 `semantic_cache`（`etc/semantic_cache_collection.json`）
  - 主键 `cache_id`（对应 MySQL `chat_cache_entry.id`）；`vector`（1024 维）+ `query_text`（改写后问题）
  - 范围过滤字段 `chat_mode / scope_document_ids / scope_task_ids / scope_knowledge_base_ids`

### MinIO 桶布局
- `agent-document/`
  - `rag/document/`：原始文档
  - `rag/parsed-text/`：解析后纯文本
  - `rag/parse-artifact/`：解析中间产物

### Neo4j 知识图谱（GraphRAG，可选依赖）
默认关闭（`Neo4j.Enabled=false`）；置 `true` 并填 `Uri / Username / Password / Database` 后启用。未启用时图谱构建与检索通道自动降级，不影响主链路。

- **图模型**（`Neo4jGraphStore`）：`(:Entity)-[:REL]->(:Entity)` 存储实体与语义关系；`(:Entity)-[:IN_COMMUNITY]->(:Community)` 存储社区归属；节点携带 `pagerank / rank_boost / degree / mention_count`、关系携带 `weight / predicate_quote / quote_texts / chunk_ids` 等可解释证据。
- **落库**：按 `document_id + task_id` 事务化全删全插（幂等重建）。
- **图谱检索**（`SearchBySeed`）：种子实体命中 → n-hop 变长路径（`maxHops` 硬限 2）→ 社区召回 → 投影证据块回流。
- **构建产物**：每篇文档的图谱构建结果（状态 `SUCCESS / FAILED / EMPTY / DEGRADED`）由图谱阶段记录，失败不阻断文档索引主任务。

### 关键配置项（`etc/config-dev.yaml`）
- `Chat.Rag`：通道开关与 TopK、融合权重、Rerank、证据预算、子问题超时
- `Chat.Rag.Graph`：GraphRAG 配置（开关、抽取批大小、实体 / 关系 / 社区上限、Louvain、PageRank、检索跳数 / TopK、图卡片投影预算等）
- `Neo4j`：图谱数据库开关与连接（可选依赖，默认关闭）
- `Chat.SemanticCache`：召回阈值、直击阈值、终判开关、置信度地板、复用策略、写入 Topic
- `Chat.Agent`：deep agent 的模型与工具调用预算、推荐开关、系统提示词
- `Chat.Memory`：压缩策略、历史轮次、字符预算
- `Embedding`：Ollama 地址、模型、维度

---

## 可观测与追踪

每一次对话都自动落库完整的可观测数据：

- `ChatDebugTrace`
  - 原始问题 / 改写问题
  - 文档导航决策（结构锚点 / 条目锚点）
  - 检索上下文、Prompt 模板、引用列表
  - 工具调用轨迹、思考步骤、模型用量与成本
- `ConversationTraceStage`
  - 按阶段（`stageCode / stageOrder / stageLevel`）记录耗时、状态、快照、错误
- `ChatChannelExecution`
  - 每个通道（向量 / 关键词 / 图谱）的召回 / 接受 / 最终选中等指标
- `ChatRetrievalResult`
  - 每条召回的原始分、RRF 分、Rerank 分、门控、选中原因
- `ChatExchangeEval`
  - 每个评估器的在线评分明细
- 语义缓存快照
  - 命中 / 未命中、相似度、置信度、命中原因、复用策略、条目 ID

常用查询入口：
- `/chat/exchange/detail`：单轮全部调试信息（排查「为什么答非所问」）
- `/chat/exchange/retrieval/results`：单轮检索结果明细
- `/chat/exchange/channel/executions`：单轮通道执行明细

---

## 接口概览

| 分组 | 前缀 | 主要接口 |
| --- | --- | --- |
| 对话 | `/chat` | `/stream`（SSE 流式对话）、`/session/stop`、`/session/detail`、`/session/list`、`/session/reset`、`/session/summary/rebuild`、`/exchange/detail`、`/exchange/retrieval/results`、`/exchange/channel/executions` |
| 文档 | `/manage/document` | `/upload`、`/page/query`、`/detail/query`、`/options`、`/delete`、`/strategy/plan/query`、`/strategy/confirm`、`/index/build`、`/chunk/query`、`/chunk/detail/query`、`/task/log/query`、`/profile/detail`、`/profile/regenerate`、`/profile/batch/regenerate` |
| 知识 | `/manage/knowledge` | `/base/list`、`/scope/save`、`/scope/delete`、`/scope/list`、`/topic/save`、`/topic/delete`、`/topic/list`、`/topic/document/*`、`/route/trace/page/query` |

知识库更新接口携带 `graphRagConfigJson`（图谱 RAG 配置）、`retrievalConfigJson`、`metadataFilterJson` 配置字段；图谱构建与图谱检索在 `Neo4j.Enabled=true` 后生效（`raptorConfigJson` 为预留字段，暂未启用）。

完整 OpenAPI 定义见 `doc/chat.json`、`doc/document.json`、`doc/knowledge.json`。

---

## 快速开始

### 1. 准备基础设施
MySQL、Redis、MinIO、Milvus、RocketMQ 需自行部署（仓库内的 `docker-compose.yml` 只编排前端与后端镜像）。图谱检索（GraphRAG）为可选能力，需另行部署 Neo4j 并在 `Neo4j.Enabled=true` 后启用。

### 2. 初始化数据库
```bash
# Linux / macOS
mysql -h<host> -uroot -p rag < etc/schema.sql
```
```powershell
# Windows PowerShell
Get-Content etc/schema.sql -Raw | mysql -h<host> -uroot -p rag
```

### 3. 修改配置
编辑 `etc/config-dev.yaml`：数据库与中间件地址、`Embedding`（Ollama 地址与模型）、`ChatModel` / `JudgeModel`（Ark 或 Ollama）、`Milvus.Collection`、`MQ` 主题、`Chat.SemanticCache` 等。

### 4. 启动后端
```bash
go run ./cmd -f etc/config-dev.yaml
```

### 5. 启动前端
```bash
cd front
npm install
npm run dev
```
### 6. 容器化部署（可选）
```bash
docker-compose up -d      # 构建并启动 frontend + backend
```

## 许可证

本项目基于 **Apache License 2.0** 开源，详见 [LICENSE](./LICENSE)。
