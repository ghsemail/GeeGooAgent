# GeeGooAgent 上下文压缩设计

> 日期：2026-07-12  
> 状态：已实现（feat/context-compression）  
> 参考：[Hermes 上下文压缩与缓存](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/context-compression-and-caching)  
> 原则：对齐 Hermes **算法与配置语义**；不照搬代码；不做 IM Gateway 双层、不做 Anthropic `cache_control`、不做可插拔 ContextEngine 插件。

## 1. 目标与非目标

### 目标

- 长对话在接近模型上下文窗时自动压缩，避免 API context-length 失败与费用失控。
- 压缩结果写回会话（SQLite / 内存 Session），后续轮次直接使用压缩后历史。
- chat CLI 与 HTTP `agent-runtime` 共用同一压缩路径（挂在 ReAct / `Agent.Run` 前）。
- 摘要使用**可配置辅助模型**；未配置时回退主 `llm`。

### 非目标（本阶段不做）

- IM Gateway / 多平台会话清理进程（无 Telegram/Discord）。
- Anthropic prompt caching 断点。
- `ContextEngine` 插件 ABC / 无损 LCM。
- 压缩中间轮次另存 archive 表（已选方案 A：直接写回，不保留完整中间原文）。
- 确定性 `workflow` / `pre_market` 路径改造（不受影响）。
- 盘中 / 盘后 skill、scheduler 常驻（后续子项目）。

> 对齐更新（2026-07-12）：在 **无 IM Gateway** 前提下，于每轮 `RunTurn` 开头增加 Hermes 同语义的 **85% session hygiene**（`hygiene_threshold`）；`context_length` 可按当前模型自动解析。

## 2. 已确认决策

| 项 | 选择 |
|---|---|
| 触发 | 按 token：`prompt_tokens ≥ threshold × context_length` |
| 摘要模型 | 可配置 `auxiliary.compression`；空则回退主 LLM |
| 持久化 | 写回会话存储（方案 A） |
| 摘要失败 | **跳过本次压缩**，保留完整中间轮（比 Hermes「丢中间」更保守） |
| 实现路径 | Hermes 精简移植（单层 Agent 压缩） |

## 3. 架构与数据流

```text
ReActLoop.RunTurn
  │
  ├─ turn-start hygiene（默认 85%）：粗估/上次 prompt_tokens ≥ hygiene_threshold × context_length
  │     └─ 同四阶段 CompressHygiene → emit context_hygiene
  │
  └─ 每轮 gateway.Chat 之前
        ├─ tokenEstimate = lastPromptTokens 或 EstimateTokens(messages)
        ├─ if !ShouldCompress（默认 50%）→ 直接 Chat
        ├─ Phase1…4 Compress → emit context_compressed
        └─ gateway.Chat…
```

回合结束后仍走现有 `SyncFromRuntime` + `SessionStore.Save`，压缩后的消息自然落盘。

## 4. 配置

`config.json` 新增（与现有 `llm` / `search` 并列）：

```json
{
  "compression": {
    "enabled": true,
    "threshold": 0.5,
    "hygiene_threshold": 0.85,
    "target_ratio": 0.2,
    "protect_last_n": 20,
    "protect_first_n": 3,
    "context_length": 0,
    "clear_tool_min_chars": 200
  },
  "auxiliary": {
    "compression": {
      "provider": "",
      "model": "",
      "token_key": "",
      "base_url": ""
    }
  }
}
```

| 字段 | 默认 | 含义 |
|---|---|---|
| `enabled` | `true` | 总开关 |
| `threshold` | `0.5` | 环内触发比例：`threshold × context_length` |
| `hygiene_threshold` | `0.85` | 回合开始安全网（对齐 Hermes Gateway 85%）；须 ≥ threshold |
| `target_ratio` | `0.2` | 尾部 token 预算 = `threshold_tokens × target_ratio` |
| `protect_last_n` | `20` | 尾部至少保留的消息条数 |
| `protect_first_n` | `3` | 头部保留（system + 首轮交互）；硬编码默认 3，可配置 |
| `context_length` | `0`（自动） | `>0` 显式覆盖；否则按当前模型解析（`llm.ResolveContextWindow`） |
| `clear_tool_min_chars` | `200` | Phase1 清除阈值 |
| `auxiliary.compression.*` | 空 | 空字段回退 `cfg.LLM` 对应项 |

校验：`threshold`/`target_ratio` 落在合理区间（与 Hermes 一致：threshold 0–1，target_ratio 0.10–0.80）；`protect_last_n ≥ 1`。

## 5. 组件与文件

| 路径 | 职责 |
|---|---|
| `internal/config/config.go` | `CompressionConfig`、`AuxiliaryConfig` 加载与默认值 |
| `internal/prompt/estimate.go` | `EstimateTokens([]Message) int`（chars/4） |
| `internal/prompt/compressor.go` | `Compressor`：ShouldCompress / Compress 四阶段 |
| `internal/prompt/summary.go` | 摘要 prompt 模板 + 调用辅助 Provider |
| `internal/prompt/compressor_test.go` | 边界、清工具、组装、失败跳过 |
| `internal/runtime/react.go` | Chat 前挂钩；记录 `lastPromptTokens` |
| `internal/agent/agent.go` | 可选：持有 `*prompt.Compressor`，构造时注入 |
| `internal/app/app.go`（或 chat/runtime 装配处） | 根据 config 构建 Compressor + 辅助 Gateway |
| `docs/architecture/overview.md` | 补一行压缩子系统说明 |
| `deploy/hermes-parity-comparison.md` | 上下文压缩改为 ✅ |

不引入 Hermes 式插件目录。

## 6. 四阶段算法（对齐 Hermes）

### Phase 1 — 清除旧工具结果（无 LLM）

对「保护尾之外」的 `role=tool` 消息，若 `len(content) > clear_tool_min_chars`，替换为：

```text
[Old tool output cleared to save context space]
```

### Phase 2 — 确定边界

```text
[0 .. protect_first_n)     头部保留
[protect_first_n .. cut)   中间 → 摘要
[cut .. end]               尾部保留
```

尾部：从末尾向前累加估算 token，直到 `tail_token_budget`；若消息数 `< protect_last_n`，则至少保留 `protect_last_n` 条。

边界对齐：不得拆开 assistant(tool_calls) ↔ 后续 tool 结果组；`alignBoundaryBackward` 跳过连续 tool，落在父 assistant 之前。

最小触发：`len(messages) < protect_first_n + protect_last_n + 1` 时不压（中间无可压内容）。

### Phase 3 — 结构化摘要

辅助 LLM（无 tools）单次调用。模板面向 GeeGoo（非编码场景）：

```text
## Goal
## Constraints & Preferences
## Progress
### Done
### In Progress
### Blocked
## Key Decisions
## Relevant Symbols / Reports
## Next Steps
## Critical Context
```

摘要 max_tokens：`clamp(content_tokens × 0.20, 2000, min(context_length × 0.05, 12000))`。

迭代重压缩：`Compressor`（或 session 旁路字段）保存 `previous_summary`；再次压缩时要求模型**更新**旧摘要而非从头写。

### Phase 4 — 组装

1. 头部消息（首次压缩可在 system 后追加一行 note：`[Note: earlier turns were compacted…]`——**仅追加 note 文本一次**，不每轮改写整段 system 人格，以免破坏前缀缓存语义；实现上优先：保留原 system 不变，把 note 放进摘要消息前缀，避免改 `Messages[0]`）。
2. 摘要消息（角色选 user 或 assistant，避免连续同 role 违规）。
3. 尾部消息未改。
4. `sanitizeToolPairs`：删孤儿 tool；缺结果的 tool_call 注入 stub。

**System 稳定性约定（覆盖旧 architecture 草稿）**：优先**不修改** `Messages[0]` 的稳定 system 正文；compaction note 放在摘要消息内。这样与 P2a「system 跨轮 byte-identical」一致。

## 7. 与现有代码的衔接

- `chatsession.RuntimeMessages` 仍可在最后一条 user 前注入 Tool 活动 context；压缩写回的是 `ChatSession.Messages` / `runtime.Session.Messages`。
- `TokenUsage.PromptTokens` 已在 `llm.Response` 中解析；ReActLoop 每轮 Chat 后写入 loop 级 `lastPromptTokens`。
- 确定性 workflow **不**调用 Compressor。
- 旧文档 `docs/architecture/layers/L3-memory/compaction.md`（L1–L4 草稿）以本 spec 为准；实现后更新该页指向本设计。

## 8. 可观测性

- Progress 事件：`context_compressed`，payload 含 `before_msgs`、`after_msgs`、`estimated_tokens_before`、`summary_chars`。
- 日志：摘要失败 warn + 原因；不把完整中间对话打进日志。

## 9. 测试计划

| 用例 | 期望 |
|---|---|
| `ShouldCompress` 低于阈值 | false |
| `ShouldCompress` 达阈值且消息足够 | true |
| Phase1 大 tool 被替换、小 tool 保留 | 内容符合 |
| 边界不拆 tool 组 | assistant+tools 同在头或同在尾 |
| 摘要成功 | 中间段消失，出现摘要消息，头尾保留 |
| 摘要失败 | 返回原 messages，无副作用 |
| 二次压缩带 previous_summary | 调用参数含旧摘要 |
| system[0] 压缩后仍等于 `chatprompt.System()` | 稳定 |

离线：mock Provider；不依赖真实 DeepSeek。

## 10. 成功标准

1. 构造超阈值假会话，一次 `Compress` 后消息数与估算 token 明显下降。
2. `go test ./internal/prompt/... ./internal/runtime/...` 通过。
3. `geegoo chat` 长对话可触发压缩 progress，且后续轮次仍可用。
4. 对比文档中「上下文压缩」标记为已实现。

## 11. 实现顺序（供 plan 拆分）

1. Config 类型 + 默认值 + 测试  
2. EstimateTokens + ShouldCompress  
3. Phase1/2/4 纯函数 + 测试（无 LLM）  
4. Phase3 摘要 + mock 测试  
5. 接入 ReActLoop + 辅助 Provider 装配  
6. 文档更新  

## 12. 风险

| 风险 | 缓解 |
|---|---|
| 粗估 token 与真实差大 | 优先用 API `prompt_tokens`；阈值 0.5 留余量 |
| 辅助模型窗口小于中间段 | 捕获错误 → 跳过压缩；可提示用户换更大摘要模型 |
| 写回后无法还原原文 | 已接受方案 A；证据链仍在 EvidenceStore / step_records |
