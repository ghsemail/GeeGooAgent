# L3 — Context / Memory

Agent 的记忆系统：会话历史、工作进度、证据链、上下文压缩。

> Go 实现：`internal/chatsession`、`internal/memory`、`internal/prompt`

## 四层模型（热 → 冷）

| 层 | 生命周期 | Go 实现 | 状态 |
|----|----------|---------|------|
| **SessionMemory** | 单次 chat turn / workflow run | `chatsession` messages | ✅ SQLite |
| **WorkingMemory** | 单次 workflow，结构化 | `memory/working.go` | ✅ |
| **Evidence** | 可审计工具结果 | `memory/evidence.go` | ✅ SQLite |
| **Episodic** | 跨日摘要 | 本地 md + jsonl（规划） | ⚠️ stub |
| **Semantic** | 向量相似检索 | — | ❌ Phase 4+ |

## 模块索引

| 文档 | 内容 | 代码 |
|------|------|------|
| [session-memory.md](./session-memory.md) | 对话持久化、FTS5 | `chatsession/sqlite.go` |
| [working-memory.md](./working-memory.md) | 盘前进度、幂等键 | `memory/working.go` |
| [compaction.md](./compaction.md) | **Hermes 风格压缩** | `prompt/compressor.go` |
| [episodic-memory.md](./episodic-memory.md) | 昨日摘要 | `recall_yesterday_summary` stub |
| [semantic-memory.md](./semantic-memory.md) | 向量检索 | 未实现 |

## SQLite Schema（L0 地基）

`internal/infra/schema.sql`：

| 表 | 用途 |
|----|------|
| `chat_sessions` | 会话元数据 |
| `session_events` | 消息与 tool 事件 |
| `evidence_records` | 工具结果审计链 |
| `working_state` | workflow 进度 |
| `checkpoints` | 步骤恢复点 |
| `execution_events` | 执行审计 |

FTS5 索引支持 `recall` 跨会话搜索。

迁移：`geegoo migrate`（文件 JSON → SQLite）。

## Prompt 稳定性（与 Hermes 对齐）

| 机制 | 实现 |
|------|------|
| 稳定 system | `chatprompt.Build()` 跨轮不变 |
| 动态 context | `RuntimeMessages()` 作 **user** 注入 |
| 压缩 | 四阶段：工具结果截断 → 摘要 → … |

详见 [compaction.md](./compaction.md)。

## Evidence 链（GeeGoo 差异化）

每条 workflow 工具结果写入 `evidence_records`（payload + hash）。

报告 `create_pre_market_report` 只存 evidence ID；`geegoo verify` 校验引用完整。

## 与 StateStore 分工

- **L3**：语义 API（Working、Evidence、Session 接口）
- **L0**：`infra/db.go` 句柄、WAL、迁移

## 外部依赖

**MVP 不需要** Agent 侧 PostgreSQL / 向量库。业务数据经 L2 调 GeeGoo HTTP。

完整决策表见下文「外部依赖决策」章节（原 README 保留）。

## 外部依赖决策（数据库 / 向量库 / Embedding）

**结论：盘前 MVP 不需要自建业务数据库、向量库或 embedding 模型。**

| 能力 | Phase 0–1 | Phase 4+ |
|------|-----------|----------|
| 会话持久化 | SQLite | 同上 |
| 跨日记忆 | 本地 md + jsonl | 同上 |
| 业务数据 | GeeGoo API | GeeGoo API |
| 语义检索 | stub | 可选 Chroma / sqlite-vss |

### 与 Hermes / Claude Code 对照

| 维度 | Hermes | GeeGooAgent |
|------|--------|-------------|
| 会话 | SQLite + FTS5 | SQLite + FTS5 ✅ |
| 跨 run 记忆 | 文件 + 日志 | `recall_yesterday_summary` ❌ stub |
| 向量库 | 无 | MVP 无 |

## 延伸阅读

- [../L1-model-gateway/gateway.md](../L1-model-gateway/gateway.md) — LLM 上下文预算
- [../L2-tools/README.md](../L2-tools/README.md) — `recall` Tool
- [../../implementation-status.md](../../implementation-status.md) — 记忆层实现状态
