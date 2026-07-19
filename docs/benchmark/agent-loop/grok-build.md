# Agent Loop：GeeGooAgent × Grok Build

> 更新：2026-07。GeeGoo 以 `internal/agent` 为准；Grok Build 以 [开源仓库](https://github.com/xai-org/grok-build)、[docs.x.ai/build](https://docs.x.ai/build/overview) 及 [grok-build 功能整理](../grok-build.md) 为准。

## 摘要

| 维度 | 结论 |
|------|------|
| **产品形态** | Grok Build = **编码 harness**（TUI + `xai-grok-shell`）；GeeGoo = **金融工具编排 Agent**（Go `Loop` + 可选 Workflow） |
| **Loop 骨架** | 同为 ReAct；Grok 在 harness 层强化 **Plan mode、并行子 Agent、沙箱执行、Headless JSON** |
| **GeeGoo 更强** | 金融 MCP 工具、Workflow/Supervisor、Evidence 审计、确定性盘前、Hermes 对齐的压缩/缓存/clarify |
| **Grok 更强** | Plan 批准门控、多路并行子 Agent、文件/终端/Git 工具链、ACP/Headless CI、Hooks、生态兼容（AGENTS.md / Cursor） |

---

## 1. 架构对照

### 1.1 Crate / 包映射

```text
Grok Build (Rust)                   GeeGooAgent (Go)
─────────────────                   ───────────────
xai-grok-pager-bin  → grok CLI      cmd/geegoo + cmd/agent-runtime
xai-grok-pager      → TUI           internal/cli/chattui + chatui
xai-grok-shell      → Agent runtime internal/agent/loop.go
xai-grok-tools      → 工具实现       internal/tools/ + MCP 客户端
xai-grok-workspace  → FS/VCS/检查点  internal/memory/working + workflow checkpoint
```

Grok 把 **harness（shell）** 与 **工具（tools）** 与 **工作区（workspace）** 拆成独立 crate；GeeGoo 用 `App` 聚合 Gateway、Registry、Workflow，loop 在 `internal/agent`。

### 1.2 运行形态

| 形态 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| 交互 TUI | ✅ 全屏 `grok` | ✅ `geegoo chat`（Bubble Tea） |
| Headless 对话 | ✅ `grok -p "..."` + streaming JSON | ⚠️ 无等价 `-p`；有 `geegoo run` / scheduler / HTTP runtime |
| ACP / IDE | ✅ | ❌ |
| 确定性批处理 | ⚠️ 仍走 Agent loop | ✅ `workflow.Runner` + `geegoo verify` |

---

## 2. Agent Loop 机制对比

### 2.1 标准 ReAct 环

两者核心均为：

```text
message + tools context
  → model (stream)
  → parse tool_calls
  → execute tools (possibly parallel)
  → append results
  → repeat until no tools or budget hit
```

| 环节 | Grok Build (`xai-grok-shell`) | GeeGooAgent (`Loop`) |
|------|-------------------------------|----------------------|
| 流式解析 | ✅ | ✅ `ChatStream` + delta 回调 |
| 并行 tool | ✅ | ✅ `tool_max_parallel`（默认 4） |
| 步数上限 | ✅ | ✅ `max_steps`（默认 80，硬顶 90） |
| 预算耗尽 | ✅ 阶段性收尾 | ✅ `finishBudgetExhausted` + 无 tool 终局 LLM |
| 用户中断 | ✅ | ✅ `context.Context` |
| 上下文压缩 | ✅（实现未完全公开） | ✅ Hermes 四阶段 + 双阈值 |
| Prompt cache | —（依赖端点） | ✅ `ApplyCacheBreakpoints` |

### 2.2 Plan mode（最大差异之一）

| | Grok Build | GeeGooAgent |
|---|------------|-------------|
| **结构化计划** | ✅ Plan mode：先出计划，**批准前禁止改代码**；默认 `.grok/plan.md` | ❌ ReAct 内无独立 Plan 阶段 |
| **Diff 审查** | ✅ 变更以 diff 呈现 | —（不写本地代码库） |
| **计划与执行分离** | harness 强制门控 | LLM 可在同轮 stream 计划文本后立即 `tool_calls`（TUI 会区分 plan vs 最终 reply） |

**解读**：Grok 的 Plan mode 是 **harness 层状态机**，不是多一个 prompt 段落。GeeGoo 金融场景若要做「Bot 创建先审后执行」，需另加 workflow 步骤或未来引入类似门控，当前 **未实现**。

### 2.3 推理与澄清

| 能力 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| Deep reasoning 展示 | ✅ 分步思考 UI | ⚠️ 模型 `reasoning` + `thinking_*` 事件 |
| 用户澄清 | ✅ ask_user / Q&A，`--no-ask-user` | ✅ `clarify` tool + TUI + HTTP `/v1/chat/clarify` |
| 流式工具参数 | ✅ | ✅ `tool_gen_start` / `tool_gen_delta` |

---

## 3. 子 Agent 与并行

| 能力 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| 委派工具 | `task` / `spawn_subagent` | `delegate_task` |
| **并行子 Agent** | ✅ 宣称多路（~8） | ❌ 顺序 |
| 独立步数预算 | ✅ | ✅ `sub_agent_max_steps`（20，最大 40） |
| 独立上下文 | ✅ | ✅ 独立 session，不写主历史 |
| 禁止嵌套 | ✅ | ✅ |
| **Git worktree 隔离** | ✅ 子 Agent 可选 worktree | — |
| 子 Agent 指定模型 | ✅ per-child `model` | ❌ |

**优劣**：多标的并行调研、多模块编码任务 Grok 更合适；GeeGoo `delegate_task` 够用但 **无并行、无 worktree**，盘前 per-stock 并行主要靠 **Workflow 代码编排** 而非 loop 内子 Agent。

---

## 4. 工具执行环境（Loop 的「Act」层）

| 工具类 | Grok Build | GeeGooAgent |
|--------|------------|-------------|
| 读/写/补丁文件 | ✅ 多文件 search-replace | ❌ |
| 终端 / Shell | ✅ 流式 bash | ❌ |
| 沙箱执行 | ✅ sandboxed execution | ❌ |
| 后台长任务 | ✅ Tasks / Watchers | — |
| Git | ✅ stage/commit/push | — |
| 代码库 Grep | ✅ | ⚠️ `search_code`（信号库语义） |
| Web search | ✅ | ✅ DuckDuckGo + 新闻 skill |
| MCP | ✅ 客户端（Linear/Sentry/…） | ✅ 消费 GeeGooBot MCP |
| 金融 MCP（行情/信号/Bot） | — | ✅ ~82 工具 |
| 写操作审批 | ✅ 权限提示 | ✅ `approval` + TUI |

Grok 的 loop **围绕本地工作区转**；GeeGoo 的 loop **围绕远程 API 与 MCP 转**，故意不暴露 arbitrary shell。

---

## 5. 上下文、记忆与检查点

| 能力 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| 会话持久化 | ✅ Memory 跨会话 | ✅ SQLite `chatsession` |
| 工作区检查点 | ✅ `xai-grok-workspace` | ✅ workflow `checkpoint` + `working` 状态 |
| AGENTS.md / 目录规则 | ✅ + CLAUDE.md 兼容 | ⚠️ `rules/` + `prompts/`，非 AGENTS.md |
| Hooks（工具/编辑事件） | ✅ | ❌ |
| Evidence 哈希审计 | — | ✅ `evidence_records` |

GeeGoo **Workflow 检查点** 服务于 resume/supervisor；Grok **workspace 检查点** 服务于编码回滚与多子 Agent 隔离——目标不同。

---

## 6. Headless 与自动化

| 能力 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| 单命令非交互对话 | ✅ `grok -p` | ❌ |
| 结构化输出 | ✅ `--output-format streaming-json` | ⚠️ EventBus / 日志；`verify` 为验收非对话 |
| CI 集成 | ✅ 官方定位 | ⚠️ `geegoo run` + `geegoo verify` + scheduler |
| Cron | —（用户自建） | ✅ `geegoo scheduler run` + supervisor 退避重试 |

若要把 **自然语言 Agent turn** 嵌入 CI，Grok Headless 更成熟；GeeGoo 自动化应优先 **Workflow + verify**，而非模拟 `grok -p`。

---

## 7. 可观测性（Loop 执行期）

| 维度 | Grok Build | GeeGooAgent |
|------|------------|-------------|
| TUI 进度 | 全屏滚动 + diff + Plan 审查 | Hermes 风格 process 区 + 状态栏 |
| 配置发现 | `grok inspect` | `geegoo inspect` + `geegoo doctor`（连通性） |
| 子 Agent 事件 | ✅ | ✅ `subagent_*` |
| Token / 缓存 | 端点依赖 | ✅ status bar + `prompt_cache` 事件 |

---

## 8. 优劣总结

### 8.1 GeeGooAgent 相对 Grok Build（Agent Loop）

**优势**

1. **领域 loop**：行情、信号、Bot、报告工具一等公民，非编码套壳。
2. **Workflow 第二轨道**：盘前不依赖 LLM 逐步决策；Supervisor + `geegoo verify`。
3. **Evidence 与报告合成**：loop 产出可审计、结论字段可规则锁定。
4. **Hermes 对齐能力**：压缩、缓存断点、clarify、delegate、审批、预算总结已落地。
5. **部署**：单 Go 二进制 + 自有 scheduler；不绑定 xAI 订阅。

**劣势**

1. 无 Plan mode 批准门控。
2. 无并行子 Agent / worktree。
3. 无文件/终端/Git loop 工具。
4. 无 Headless `-p` 式对话 CLI。
5. 无 ACP / Hooks / 插件市场。
6. 无 `grok inspect` 级扩展发现。

### 8.2 Grok Build 相对 GeeGooAgent（Agent Loop）

**优势**

1. **Harness 完整度**：Plan、沙箱、并行子 Agent、后台任务一体化。
2. **编码 loop 工具链**：移植 Codex/OpenCode 工具实现，成熟度高。
3. **Headless + ACP**：CI 与 IDE 同一 loop。
4. **生态兼容**：AGENTS.md、Cursor/Claude skills/hooks/MCP。
5. **Rust 性能与类型安全**（对长会话/大工具输出友好）。

**劣势（相对 GeeGoo 金融场景）**

1. 无盘前 Workflow / Supervisor / 业务 verify。
2. 无金融 MCP 工具域。
3. 开源树不接受外部 PR；模型集成部分在 monorepo 闭源。
4. Cron/批处理需自行编排，无一等公民 scheduler+verdict 重试。

---

## 9. 可借鉴项（GeeGoo 路线图参考）

按投入产出排序，**仅 Agent Loop 相关**：

| 优先级 | Grok 能力 | GeeGoo 现状 | 建议 |
|--------|-----------|-------------|------|
| P1 | Plan mode + 批准门控 | ❌ | 复杂 Bot/写操作可先出计划再执行（workflow 或 chat 门控） |
| P2 | Headless JSON 流 | ⚠️ | HTTP runtime 或 `geegoo chat --message` 统一 streaming 事件 schema |
| P3 | `inspect` 式发现 | ✅ `geegoo inspect` | doctor 偏连通性；inspect 展示 loop/toolsets/skills |
| P4 | 并行 `delegate_task` | ❌ | 多标的并行分析（注意 MCP 限流） |
| P5 | Hooks | ❌ | 合规审计（tool 前后脚本） |
| — | 文件/终端工具 | 不做 | 与产品定位冲突 |

---

## 10. 选型建议

| 场景 | 建议 |
|------|------|
| 终端里改代码、跑测试、开 PR | Grok Build |
| 盘前报告、信号查询、Bot 管理、MCP | GeeGooAgent |
| CI 里跑「一句话修 bug」 | Grok `grok -p` |
| CI 里验收盘前报告字段 | GeeGoo `geegoo verify` |
| 同一 harness 兼顾两者 | 不现实；保持 **GeeGoo 金融 loop + 必要时外挂 Grok 做仓库维护** |

---

## 11. 参考

- Grok 功能清单：[../grok-build.md](../grok-build.md)
- 三方速查：[../comparison.md](../comparison.md)
- GeeGoo loop 实现：[../../architecture/layers/L4-runtime/agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md)
- Grok 仓库：<https://github.com/xai-org/grok-build>
