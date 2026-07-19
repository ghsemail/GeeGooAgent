# Agent Loop：GeeGooAgent × Hermes Agent

> 更新：2026-07。GeeGoo 以仓库 `internal/agent` 与 `geegoo verify agent-loop` 为准；Hermes 以 [官方架构](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture) 与 [hermes-parity-comparison.md](../../../deploy/hermes-parity-comparison.md) 为准。  
> **线上**：119.45.16.112 已部署 `225615ed`（2026-07-19）；`geegoo inspect --quick` 9/9 PASS。

## 摘要

| 维度 | 结论 |
|------|------|
| **循环骨架** | 两者均为标准 ReAct：`LLM(+tools) → tool_calls? → 执行 → 写回 session → 重复` |
| **核心对齐度** | GeeGoo **已对齐** Hermes agent-loop 主路径（压缩、缓存断点、delegate、clarify、审批、预算耗尽总结、可中断） |
| **最大结构差异** | GeeGoo 在 ReAct 之外另有 **确定性 Workflow**（`geegoo run` / scheduler），Hermes cron 多为「新建 Agent + skill prompt」 |
| **GeeGoo 更强** | Workflow Supervisor、Evidence 审计、报告合成防失控、`geegoo verify` 量化验收 |
| **Hermes 更强** | 18+ Provider / 3 种 API mode、插件与 Context Engine、70+ 工具含终端/浏览器、Gateway/ACP、轨迹导出 |

---

## 1. 架构对照

### 1.1 入口与核心

```text
Hermes                              GeeGooAgent
────────                            ───────────
cli.py ──────────┐                  geegoo chat ─────────┐
gateway (20 IM) ─┼→ AIAgent        agent-runtime HTTP ───┼→ Agent.Run → Loop.RunTurn
cron / ACP ──────┘   run_agent.py   geegoo run / scheduler ┘→ App.RunSkill → workflow.Runner
```

Hermes 强调 **单一 `AIAgent`** 服务 CLI、Gateway、Cron、ACP。GeeGoo 同样坚持 **平台差异在入口**（见 [entrypoints.md](../../architecture/entrypoints.md)），但把 **定时盘前** 从 ReAct 中拆出为 Workflow，避免 LLM 漏步。

### 1.2 代码映射

| Hermes (Python) | GeeGooAgent (Go) | Loop 相关 |
|-----------------|------------------|-----------|
| `run_agent.py` — `AIAgent` | `internal/agent/agent.go` — `Agent.Run` | 统一门面 |
| `AIAgent.run_conversation()` | `internal/agent/loop.go` — `Loop.RunTurn` | 主循环 |
| `model_tools.py` | `internal/agent/tool_exec.go` + `internal/tools/registry.go` | 工具派发 |
| `agent/prompt_builder.py` | `internal/chatprompt/builder.go` | 稳定 system |
| `agent/context_compressor.py` | `internal/prompt/compressor.go` | 四阶段压缩 |
| `agent/prompt_caching.py` | `internal/llm/cache.go` | 缓存断点 |
| `tools/delegate_tool.py` | `internal/agent/delegate_tool.go` | 子 Agent |
| `tools/clarify_tool.py` | `internal/tools/clarify.go` | 用户澄清 |
| `tools/approval.py` | `internal/tools/approval.go` | 写操作审批 |
| — | `internal/workflow/supervisor.go` | **GeeGoo 特有**（非 loop 内，绑 workflow） |

---

## 2. 单次 Turn 生命周期

### 2.1 共同流程

```text
用户输入
  → 追加 user message
  → [可选] 回合开始 hygiene 压缩（约 85% 上下文）
  └─ for round in 1..max_steps:
        → [可选] 轮前压缩（约 50% 阈值）
        → SanitizeMessages（合并连续同角色消息）
        → Gateway.Chat / ChatStream（带 tool schemas）
        → 无 tool_calls → 返回最终 assistant 文本
        → 有 tool_calls → ExecuteBatch → 追加 assistant + tool messages → 继续
  → 达 max_steps → 预算耗尽终局（再调一次无 tool 的 LLM 做阶段性总结）
```

GeeGoo 实现见 `Loop.RunTurn`（`loop.go`）与 `runRound`（`loop_round.go`）。

### 2.2 逐步对比

| 阶段 | Hermes | GeeGooAgent | 评价 |
|------|--------|-------------|------|
| **System prompt** | `prompt_builder` 跨轮稳定 | `chatprompt` Soul + ToolRouting + Memory，跨轮字节稳定 | ✅ 对齐 |
| **动态上下文** | 记忆/技能/AGENTS 注入 | `RuntimeMessages()` 作 user 注入，不污染 system | ✅ 对齐 |
| **Hygiene 压缩** | Gateway Session ~85% | `applyHygiene` + `ShouldHygiene` | ✅ 对齐 |
| **轮前压缩** | 中间轮有损摘要 | `applyCompression` 四阶段 | ✅ 对齐；GeeGoo 摘要失败时 **跳过压缩**（更保守） |
| **流式** | stream + spinner | `ChatStream` + `stream_delta` / `thinking_*` / `tool_gen_*` | ✅ 对齐 |
| **畸形 tool call** | 重试策略 | `MalformedToolCallResponse` → slim schema 重试 | ✅ 对齐 |
| **并行 tool** | 支持 | `tool_max_parallel` 默认 4 | ✅ 对齐 |
| **tool 超时** | 支持 | `tool_timeout_sec` 默认 120s | ✅ 对齐 |
| **中断** | 信号/用户输入 | `context.Context` 贯穿 LLM + Tool | ✅ 对齐（Go 更直接） |
| **预算耗尽** | 返回已完成工作摘要 | `finishBudgetExhausted` + 无 tool 终局 LLM | ✅ 对齐 |

### 2.3 GeeGoo 独有（同一次「对话 turn」内）

| 机制 | 说明 |
|------|------|
| **Chat 工具拦截** | Interactive 模式从 schema 剔除 workflow 独占 tool；运行时 `read_working_state` 等 → `tool_intercepted` 引导 `recall`（`tool_intercept.go`） |
| **消息 sanitize** | 每轮 LLM 前 `llm.SanitizeMessages`，避免连续 user/assistant 破坏 tool 配对 |
| **Prompt cache 事件** | `prompt_cache` 上报 hit/miss tokens（DeepSeek/Minimax 等） |
| **离线 parity** | `geegoo verify agent-loop` 检查 clarify/delegate/recall、缓存断点、workflow 隔离、嵌套 delegate |

---

## 3. Prompt、上下文与压缩

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| Prompt 稳定性原则 | ✅ 中途不改 system | ✅ P2a 专门保证 |
| 四阶段压缩 | ✅ | ✅ `compressor.go` |
| 双阈值 | hygiene + 轮前 | 85% 回合开始 + 50% 每轮 LLM 前 |
| Context Engine 插件 | ✅ ABC，可替换 | ❌ 固定 compressor |
| 辅助 LLM | ✅ `auxiliary_client` | ❌ 无独立旁路客户端 |
| Prompt cache | Anthropic 显式为主 | `ApplyCacheBreakpoints`（system + 稳定历史边界） |
| 会话血缘 | SQLite FTS5 + 完整血缘 | SQLite FTS5 + 压缩 lineage metadata（**未完全对齐**） |

**优劣**：Hermes 在 **可插拔上下文引擎、多后端缓存策略** 上更灵活；GeeGoo 在 **金融场景下的保守压缩与稳定 system** 上更省心，但扩展新压缩策略需改代码。

---

## 4. Provider 与 Gateway（影响 loop 的外围）

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| Provider 数量 | 18+ | 3（DeepSeek / OpenAI / Minimax） |
| API mode | chat_completions / codex_responses / anthropic_messages | 仅 chat_completions |
| Fallback | ✅ | ✅ `SetFallbacks` 有序备份 |
| OAuth / 凭据池 | ✅ | 简化 token + base_url |

Loop 本身不依赖 provider 数量；Hermes 适合多模型混用与 Codex/Anthropic 原生 API，GeeGoo 适合固定几家 OpenAI 兼容端点的生产部署。

---

## 5. 工具执行与子 Agent

### 5.1 工具层

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| 注册表规模 | 70+，28 toolset | ~82，domain toolset |
| 自注册 | 导入时 `registry.register()` | `init()` + catalog/bespoke/agent |
| 审批 | `approval.py` | TUI y/n + `approval.go` |
| clarify | `clarify_tool.py` | `clarify.go` + TUI 选项 + HTTP runtime（`POST /v1/chat/clarify`） |
| Schema 校验 | jsonschema 较完整 | Meta + 空成功检测，**无完整 jsonschema** |
| 终端 / 浏览器 | ✅ 多 backend | ❌ 刻意不做（非 coding agent） |
| MCP | 客户端动态 | 消费 GeeGooBot MCP（:3120） |

### 5.2 delegate_task

| 能力 | Hermes | GeeGooAgent |
|------|--------|-------------|
| 工具名 | `delegate_task` | 同名 |
| 独立步数预算 | ✅ | `sub_agent_max_steps`（默认 20，最大 40） |
| 独立会话 | ✅ | ✅ 不写主会话历史 |
| 禁止嵌套 | ✅ | ✅ `DelegateDepth >= 1` 拒绝 |
| 并行子 Agent | ⚠️ 有限 | ❌ 顺序执行 |

---

## 6. 可观测性与回调

Hermes 通过 CLI spinner / gateway 消息暴露工具进度。GeeGoo 对齐事件包括：

| 事件 | 用途 |
|------|------|
| `turn_start` / `round_start` | 回合与轮次边界 |
| `stream_delta` | 助手正文打字机 |
| `thinking_start` / `thinking_stop` | 推理分段（模型 reasoning） |
| `tool_gen_start` / `tool_gen_delta` | 工具参数流式预览 |
| `llm_plan` / `llm_tools` | 计划与工具决策 |
| `tool_start` / `tool_done` | 工具执行 |
| `budget_exhausted` / `budget_warning` | 步数预算 |
| `prompt_cache` | 缓存命中统计 |
| `subagent_*` | 子 Agent 生命周期 |

此外 GeeGoo EventBus 发 `TurnStarted` / `TurnCompleted` / `TurnFailed`（workflow 路径另有 `RunStarted` 等）。

---

## 7. 设计原则对照（Hermes 六原则）

| 原则 | Hermes | GeeGooAgent |
|------|--------|-------------|
| Prompt 稳定性 | ✅ | ✅ |
| 可观测执行 | ✅ | ✅ EmitProgress + TUI |
| 可中断 | ✅ | ✅ ctx |
| 平台无关核心 | ✅ 单一 AIAgent | ✅ `Agent.Run` |
| 松耦合 | ✅ registry + check_fn | ⚠️ MCP/Search 等仍部分硬编进 Deps |
| Profile 隔离 | ✅ 独立 HERMES_HOME | ✅ `profiles` + `GEEGOO_PROFILE` |

---

## 8. 优劣总结

### 8.1 GeeGooAgent 相对 Hermes（Agent Loop 视角）

**优势**

1. **双轨编排**：chat 用 ReAct，盘前用 Workflow + Supervisor，cron 不赌 LLM 记步骤。
2. **Evidence + 合成约束**：loop 产出的工具结果可审计；报告层防 LLM 翻转决策（workflow 路径）。
3. **验收闭环**：`geegoo verify agent-loop` 离线 parity + `geegoo verify` 业务字段矩阵。
4. **运行时**：Go 单二进制、`context` 取消路径清晰。
5. **Chat/Workflow 工具隔离**：减少 interactive 误调 `read_working_state` 等。

**劣势**

1. Provider / API mode 面窄。
2. 无 Context Engine / 记忆插件 ABC。
3. 无轨迹训练导出（`trajectory.py` 等价物）。
4. Headless `geegoo chat -p` 尚未对齐 Grok `-p`（可用 `--message` + `--output-format ndjson`）。
5. 工具 schema 校验为轻量 JSON-schema 子集（非完整 jsonschema）。
6. 松耦合程度仍不如 Hermes 插件生态。

### 8.2 Hermes 相对 GeeGooAgent（Agent Loop 视角）

**优势**

1. 单一 ReAct 覆盖 CLI / IM / Cron / ACP，心智模型统一。
2. 工具面广（shell、浏览器、文件、MCP 生态）。
3. Provider 与 API mode 最全。
4. 插件、hooks、记忆提供者、Context Engine 可替换。
5. 测试与社区体量大（3000+ pytest）。

**劣势（相对 GeeGoo 金融场景）**

1. 无内置 Workflow Supervisor / evidence-only 报告合成。
2. Cron 任务依赖 Agent 自主编排，盘前幂等与逐步验收弱。
3. 无 `geegoo verify` 式业务 cutover 矩阵。

---

## 9. 选型建议

| 场景 | 建议 |
|------|------|
| 通用 IM / 编码 / 多 Provider Agent 平台 | Hermes |
| 股票盘前 cron + 可 resume + 可验收 | GeeGoo Workflow + scheduler（ReAct 作补充） |
| 交互式问股、Bot 管理、MCP 工具 | GeeGoo `geegoo chat`（与 Hermes CLI 同类） |
| 从 Hermes `geegoo` cron 迁移 | GeeGoo 核心 loop 已对齐；切换重点是 **workflow 路径** 而非重写 ReAct |

---

## 10. 参考

- GeeGoo loop 设计：[agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md)
- Hermes 对齐交付：[hermes-parity-comparison.md](../../../deploy/hermes-parity-comparison.md)
- 验收命令：`geegoo verify agent-loop --config ~/.geegoo/config.json`
