# GeeGooAgent vs Hermes Agent — 结构与功能对比

> 对照 [Hermes Agent 架构](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture)。
> 本文档存档 P1–P8 完成后的对齐情况，供后续决策参考。
> 路线图见 [`hermes-parity-roadmap.md`](hermes-parity-roadmap.md)，cutover runbook 见 [`hermes-migration-checklist.md`](hermes-migration-checklist.md)。

## 一、目录结构对比

| Hermes (Python) | GeeGooAgent (Go，当前) | 状态 |
|---|---|---|
| `run_agent.py` AIAgent | `internal/agent/agent.go` `Agent.Run` | ✅ 对齐（门面） |
| `agent/prompt_builder.py` | `internal/chatprompt/builder.go` + `soul.go` / `tool_routing.go` / `memory.go` + `RuntimeMessages` 动态 context | ✅ 分层 builder |
| `agent/context_compressor.py` | `internal/prompt/compressor.go` + `summary.go` | ✅ 对齐（Hermes-style 四阶段） |
| `agent/prompt_caching.py` | `internal/llm/cache.go` + 稳定 system / `RuntimeMessages` | ✅ 显式断点 |
| `hermes_cli/runtime_provider.py` (18+ provider) | `internal/llm/presets.go` (DeepSeek/OpenAI/Minimax) + `BuildProviderFromLLMFields` | ✅ 按需精简 |
| `tools/registry.py` (70+/28 toolset) | `internal/tools/registry.go` + `catalog/` + `bespoke.go` + `domains.go` | ✅ 对齐 |
| `tools/approval.py` | `internal/tools/approval.go` | ✅ 对齐 |
| `hermes_state.py` SQLite+FTS5+血缘 | `internal/chatsession/sqlite.go` + `infra/db.go` (WAL+FTS5) | ✅ 对齐（血缘未做） |
| `gateway/session.py` | `internal/chatsession/` | ✅ |
| `gateway/platforms/` (20 IM) | — | ❌ 不做（不需要 IM） |
| `cron/jobs.py` | `internal/scheduler/` (robfig/cron) | ✅ 对齐 |
| `plugins/memory/` `plugins/context_engine/` | — | ❌ 不做（单租户 YAGNI） |
| `acp_adapter/` | — | ❌ 不做 |
| `skills/` `optional-skills/` | `skills/` 资源 + `internal/skills/` Go 加载器 | ✅ 对齐 |
| `agent/trajectory.py` | — | ❌ 未做（可选） |
| — | `internal/workflow/supervisor.go` | ✅ GeeGoo 特有（Hermes 无显式 supervisor） |
| — | `internal/report/synthesis.go` | ✅ GeeGoo 特有（evidence-only LLM 综合） |
| — | `internal/verify/verify.go` | ✅ GeeGoo 特有（cutover 量化验收） |
| — | `internal/search/` (DuckDuckGo) | ✅ GeeGoo 特有（免费网页搜索） |

## 二、功能对比

| 能力 | Hermes | GeeGooAgent | 对齐度 |
|---|---|---|---|
| Agent 循环 | AIAgent 单一入口 | `Agent.Run(ctx,sess,input)` | ✅ |
| Prompt 稳定性 | ✅ system 不变 | ✅ P2a 修复 | ✅ |
| 上下文压缩 | ✅ 有损摘要 + Gateway 85% hygiene | ✅ 四阶段 + 回合前 85% hygiene + 按模型 context_length | ✅ |
| Prompt 缓存断点 | ✅ Anthropic 显式 | ✅ `ApplyCacheBreakpoints` + `cache_control`（DeepSeek/Minimax 默认开启） | ✅ |
| Provider 数量 | 18+ | 3 (DeepSeek/OpenAI/Minimax) | 按需精简 |
| API mode | 3 种 (chat/codex/anthropic) | 1 种 (chat_completions) | 按需精简 |
| 工具数 | 70+ | ~82 (catalog+bespoke) | ✅ |
| 工具自注册 | ✅ 导入时 | ✅ `init()` + `AddRegistrar`（catalog/bespoke/agent） | ✅ |
| toolset 分组 | ✅ 28 toolset | ✅ `tools/toolset.go` + `chat_toolsets` + `/toolsets` | ✅ |
| 危险操作审批 | ✅ approval.py | ✅ approval.go | ✅ |
| 工具契约/schema | ✅ | ⚠️ Meta + 空成功检测，无 jsonschema | ⚠️ |
| 会话持久化 | SQLite+FTS5+血缘 | SQLite+FTS5+压缩血缘（metadata） | ✅ |
| 迭代预算耗尽 | 返回已完成工作摘要 | ✅ `finishBudgetExhausted` + 无 tool 终局 LLM 调用 | ✅ |
| 子 Agent / delegate | ✅ delegate_task | ✅ 独立会话 + `sub_agent_max_steps`；禁止嵌套 | ✅ |
| Chat 工具拦截 | ✅ agent 层 memory/todo 类 | ✅ workflow 独占 tool 过滤 + `tool_intercepted` | ✅ |
| 平台无关核心 | ✅ | ✅ P2c | ✅ |
| 松耦合 | ✅ 注册表+check_fn | ⚠️ Deps 硬编 | ⚠️ |
| Profile 隔离 | ✅ | ✅ `profiles` + `GEEGOO_PROFILE` | ✅ |
| Cron | ✅ agent 一等公民 | ✅ P7 robfig/cron | ✅ |
| 失败重跑 | ✅ | ✅ supervisor verdict 驱动退避重试 | ✅ |
| Supervisor 质检 | ❌ 无显式 | ✅ P3 verdict pass/recoverable/terminal | GeeGoo 更强 |
| 幂等 resume | ✅ | ✅ P3 按 step key | ✅ |
| 报告生成 | 通用 LLM | ✅ P5 evidence-only 综合 + 规则 result/confidence | GeeGoo 更克制 |
| Evidence 可追溯 | ❌ | ✅ P1 EvidenceStore SQLite + hash | GeeGoo 更强 |
| Cutover 量化验收 | ❌ | ✅ P8 `geegoo verify` | GeeGoo 更强 |
| 网页搜索 | ✅ web_tools | ✅ DuckDuckGo（免费） | ✅ |
| IM 平台 | ✅ 20 | ❌ 不做 | 按需 |
| ACP/IDE 集成 | ✅ | ❌ 不做 | 按需 |
| 插件系统 | ✅ | ❌ 不做 | YAGNI |
| 轨迹训练导出 | ✅ | ❌ 不做 | 不需要 |

## 三、设计原则对齐

| Hermes 原则 | GeeGooAgent |
|---|---|
| Prompt 稳定性 | ✅ P2a |
| 可观测执行 | ✅ EmitProgress + UI |
| 可中断 | ✅ P2b |
| 平台无关核心 | ✅ P2c |
| 松耦合 | ⚠️ 部分（MCP/Search 仍硬编进 Deps） |
| Profile 隔离 | ✅ |

## 四、GeeGooAgent 相对 Hermes 的差异化优势

1. **Supervisor 质检**：Hermes 没有，GeeGoo 跑完自动 verdict，recoverable 自动补跑
2. **Evidence 可审计**：报告每条结论可追溯到原始工具 payload + hash，Hermes 无
3. **LLM 报告防失控**：result/confidence 锁定规则，LLM 只写综合文字，不能翻转决策
4. **量化 cutover 验收**：`geegoo verify` 字段完整率矩阵，Hermes 无
5. **免费网页搜索**：DuckDuckGo 内建，无需 key

## 五、GeeGooAgent 仍弱于 Hermes 的地方

1. **显式 prompt 缓存断点**：已支持 `cache_control` 断点 + DeepSeek cache hit 统计；OpenAI 仍主要靠自动前缀缓存
2. **工具自注册**：`init()` 注册 catalog/bespoke/agent 扩展
3. **Profile 隔离**：`config.profiles` + `GEEGOO_PROFILE` 覆盖 output_dir/mcp_token/chat_toolsets
4. **CLI 打字机**：`stream_delta` 边生成边显示；reasoning 经 `thinking_start/stop` 分段；tool 参数经 `tool_gen_start/delta` 预览

## 六、结论

**核心 Agent 能力已对齐 Hermes**：平台无关核心、稳定 prompt、可中断、SQLite 持久化、cron 调度、技能化、报告合成、工具审批、toolset 分组、cutover 验收。

**GeeGoo 在质检/审计/防失控/验收上超越 Hermes**，因为这些是金融场景特有需求。

**仍待补（不阻塞 cutover）**：无 — Hermes agent-loop 对齐项已落地；用 `geegoo verify agent-loop` 做离线验收。

## 七、P1–P8 交付索引

| Phase | Commit | 内容 |
|---|---|---|
| P1 | `0abf256` | SQLite 地基：session/evidence/working/checkpoint 落盘，FTS5，`geegoo migrate` |
| P2a | `638d239` | Prompt 稳定性：system 跨轮不变，保 DeepSeek 前缀缓存 |
| P2b | `3a50acd` | ctx 贯穿 + Ctrl+C 中断 |
| P2c | `3270338` | `internal/agent` 平台无关核心，chat+runtime 走 `Agent.Run` |
| P3 | `2cd380e` | supervisor verdict + 幂等 resume + recoverable 重试 |
| P4 | `28066a3` | skill registry，`geegoo run <skill>` / `skills list` |
| P5 | `864f387` | LLM 报告综合（evidence-only，result/confidence 仍规则） |
| P6 | `3c6fcba` | tool 契约：Meta、空成功检测、approval gate、fixture replay |
| P7 | `8913a6d` | Go 内 scheduler（cron + supervisor 触发退避重试） |
| P8 | `b993792` | `geegoo verify` 字段完整率 + checklist 量化验收 |
