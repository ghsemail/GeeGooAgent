# L3 — Context / Memory

Agent 的记忆系统，分四层热→冷。

## 模块设计说明

L3 回答「Agent 记得什么、记多久、怎么塞进 LLM 上下文」。与 L0 StateStore 分工：**L3 定义语义与 API，L0 负责落盘**；Memory 模块不直接操作文件路径，而通过 StateStore 抽象持久化。

**四层模型（热 → 冷）**

| 层              | 生命周期       | 内容                   | MVP  |
| -------------- | ---------- | -------------------- | ---- |
| SessionMemory  | 单次 run     | 消息历史、tool 结果、step 计数 | ✓    |
| WorkingMemory  | 单次 run，结构化 | 阶段 A/B 进度、每股中间态、幂等键  | ✓    |
| EpisodicMemory | 跨日         | 昨日报告摘要、态度轨迹 jsonl    | ✓    |
| SemanticMemory | 长期         | 向量检索相似盘面（Phase 4+）   | stub |

**核心设计决策**

| 决策                            | 理由                                              |
| ----------------------------- | ----------------------------------------------- |
| WorkingMemory 与 Session 分离    | LLM 上下文是「叙事」；Working 是「程序状态」，供 Supervisor 与幂等检查 |
| Compaction 在步间/阶段末            | 指数分析 5 份 + 新闻易超长；压缩保留结论、丢弃 raw JSON             |
| Episodic 文件型（MVP）             | 盘前只需「昨日同股摘要」；不上向量库降低 Phase 1 复杂度                |
| `read_working_state` 暴露为 Tool | Planner 可显式查询进度，避免重复拉取已完成的指数                    |

**与 Runtime 协作**

```text
ContextBuilder (L4)
    ├── SessionMemory.messages  → 拼 LLM messages
    ├── WorkingMemory.snapshot  → 系统提示中的进度块
    └── EpisodicMemory.recall    → 「昨日预判」段落

Executor 每步 ──▶ WorkingMemory.apply(tool_result)
Checkpoint ──▶ StateStore 持久化 session + working
```

**边界**

- **提供**：读写 API、压缩策略、召回接口
- **不提供**：Checkpoint 格式（L0）、报告全文存储路径（L2 `save_local_report` + L5 模板）

**MVP 范围**

Session + Working + Episodic（文件）；Semantic 返回空；Compaction 仅截断超长 tool 结果。

## 模块索引

| 层              | 文档                                         | 代码                     |
| -------------- | ------------------------------------------ | ---------------------- |
| SessionMemory  | [session-memory.md](./session-memory.md)   | `memory/session.py`    |
| WorkingMemory  | [working-memory.md](./working-memory.md)   | `memory/working.py`    |
| EpisodicMemory | [episodic-memory.md](./episodic-memory.md) | `memory/episodic.py`   |
| SemanticMemory | [semantic-memory.md](./semantic-memory.md) | Phase 4+               |
| Compaction     | [compaction.md](./compaction.md)           | `memory/compaction.py` |

## 与 StateStore 关系

- Memory 层定义**语义**与读写 API
- **持久化**统一走 L0 `StateStore`（FileStateStore → SQLite）

## 外部依赖决策（数据库 / 向量库 / Embedding）

**结论：盘前 MVP 不需要自建业务数据库、向量库或 embedding 模型。** Agent 是「无状态编排器 + 本地工件归档」；重数据经 L2 调 GeeGoo HTTP API（其服务端自有 MongoDB 等，Agent **不直连**）。

### 分阶段需要什么

| 能力              | Phase 0–1（MVP）         | Phase 2–3 | Phase 4+                 | 由谁提供               |
| --------------- | ---------------------- | --------- | ------------------------ | ------------------ |
| 会话 / 进度持久化      | `FileStateStore`（JSON） | 同上        | 可换 `SQLiteStateStore`    | L0 StateStore      |
| 跨日情节记忆          | 本地 `md` + `jsonl`      | 同上        | 同上                       | L3 Episodic        |
| 业务数据（报告、Bot、行情） | GeeGoo API               | GeeGoo API  | GeeGoo API                 | 远端 GeeGoo 服务         |
| 语义检索            | **不实现**（stub 返回 `[]`）  | 仍可不实现     | `SemanticMemory`         | L3，可选              |
| 向量库             | **不需要**                | **不需要**   | 可选 Chroma / sqlite-vss 等 | Agent 侧，仅 Phase 4+ |
| Embedding 模型    | **不需要**                | **不需要**   | 本地 bge / 托管 API          | 仅给归档报告建索引          |

### 盘前实际用到的「记忆」路径

```text
昨日报告摘要   → Episodic 读本地 {date}/{code}-premarket.md
当日进度       → WorkingMemory + L0 Checkpoint
幂等 / 查报告  → L2 get_stock_daily_reports（3120，GeeGoo 服务端库）
```

### 与 Hermes / Claude Code 对照

| 维度         | Hermes        | Claude Code       | GeeGoo Agent              |
| ---------- | ------------- | ----------------- | ----------------------- |
| 会话状态       | 隐式 / 日志       | IDE 会话            | StateStore + Checkpoint |
| 跨日记忆       | 本地 md、执行日志    | 规则文件（CLAUDE.md 等） | Episodic（文件）            |
| 语义检索       | 无             | 无（Grep/Read 文本搜索） | Semantic（Phase 4+，可选）   |
| Agent 侧数据库 | 无             | 无                 | MVP 无；后期 SQLite 可选      |
| 向量库        | 无             | 无                 | MVP 无                   |
| 业务数据       | GeeGoo MCP HTTP | 用户仓库文件            | GeeGoo API（远端库）           |

**Hermes**：Skill + cron prompt + 对话上下文；跨 run 靠日志与报告文件，无向量层。  
**Claude Code**：上下文即记忆；跨会话靠显式文档 + Grep，不做 embedding。  
**GeeGoo Agent**：MVP 与二者同级——**文件 + API 即可**；Semantic 层为后期「相似盘面召回」预留，避免 Phase 1 过度工程化。

### Phase 4+ 向量方案（仅当需要 `recall_similar_setup`）

| 方案      | 组件                                     | 适用         |
| ------- | -------------------------------------- | ---------- |
| A. 最轻   | 本地 md + 关键词 / Grep（仿 Claude Code）      | 归档报告 < 数百份 |
| B. 轻量向量 | sqlite-vss 或单机 Chroma + 本地 `bge-small` | 单机、隐私      |
| C. 托管   | Embedding API + pgvector               | 多机 / 云端    |

Embedding **只索引自家归档报告**，不对全市场行情建向量库——行情继续走 GeeGoo API。详见 [semantic-memory.md](./semantic-memory.md)。

## MVP

Session + Working + Episodic（文件型）；Semantic stub。