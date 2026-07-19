# GeeGooAgent × Hermes Agent × Grok Build 对比

> 更新：2026-07-20。GeeGoo 实现状态以 [implementation-status.md](../architecture/implementation-status.md) 为准；Hermes 以 [官方架构文档](https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture) 与 [hermes-parity-comparison.md](../../deploy/hermes-parity-comparison.md) 为准；Grok Build 以 [grok-build.md](./grok-build.md) 为准。

## 图例

| 符号 | 含义 |
|------|------|
| ✅ | 已具备且生产可用 |
| ⚠️ | 部分实现、降级或场景受限 |
| ❌ | 未实现 / 明确不做 |
| — | 不适用（非该产品目标场景） |

## 一、产品定位

| 维度 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| **语言** | Go | Python | Rust |
| **开源** | ✅ | ✅ | ✅（不接受外部 PR） |
| **核心场景** | 金融盘前/盘中/盘后、MCP 报告与 Bot | 通用 Agent + 多 IM 平台 | 终端编码、全栈开发工作流 |
| **主入口** | `geegoo chat` TUI、`geegoo run <skill>` | CLI + 20+ IM 网关 | `grok` 全屏 TUI |
| **Headless / CI** | ⚠️ `--message` + `--output-format ndjson`；`verify agent-loop --offline` | ⚠️ 脚本化可行 | ✅ `grok -p` + streaming JSON |
| **IDE / ACP** | ❌ | ✅ ACP adapter | ✅ ACP |
| **领域工具** | ✅ ~82 金融 MCP 工具 | ✅ 70+ 通用工具 | ⚠️ 编码工具为主 |

## 二、Agent 运行时

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| ReAct / Agent 循环 | ✅ `internal/agent` | ✅ `AIAgent` | ✅ `xai-grok-shell` |
| **Plan / 写操作门控** | ✅ `plan_gate`（mutating 先确认） | ⚠️ approval | ✅ Plan mode（编码计划） |
| **并行子 Agent** | ✅ `delegate_tasks` | ⚠️ 有限 | ✅ ~8 路 + worktree |
| **Deep reasoning** | ⚠️ 模型 reasoning + TUI `thinking_*` 事件 | ✅ | ✅ 分步思考展示 |
| 上下文压缩 | ✅ Hermes 四阶段 + 85% hygiene | ✅ | ✅（实现细节未公开） |
| Prompt 缓存断点 | ✅ `cache_control` | ✅ Anthropic 等 | — |
| Provider 数量 | 3（DeepSeek/OpenAI/Minimax） | 18+ | Grok 默认 + 自定义 OpenAI 兼容 |
| 迭代预算耗尽 | ✅ 摘要终局 | ✅ | ✅ |
| 用户澄清 | ✅ `clarify` + TUI 选项面板 | ✅ clarify_tool | ✅ ask_user / 多选 Q&A |
| 流式输出 | ✅ `stream_delta` 打字机 | ✅ | ✅ |
| 中断 | ✅ Ctrl+C / Esc | ✅ | ✅ |

## 三、子 Agent 与并行

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| 子 Agent 委派 | ✅ `delegate_task` + `SubAgent` | ✅ `delegate_task` | ✅ `task` / `spawn_subagent` |
| 并行子 Agent | ✅ `delegate_tasks` + `delegate_max_parallel` | ⚠️ 有限并行 | ✅ 宣称多路并行（~8） |
| 独立上下文 / 步数上限 | ✅ `sub_agent_max_steps`（默认 20，最大 40） | ✅ | ✅ per-child 窗口 |
| 禁止嵌套委派 | ✅ | ✅ | ✅ |
| Git worktree 隔离 | — | — | ✅ 子 Agent 可独立 worktree |
| 子 Agent 指定模型 | ❌ | ⚠️ | ✅ 可选 `model` 参数 |

## 四、Skills、规则与扩展

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| **Skills** | ✅ `skills/` + `geegoo run`；manifest 工作流包 | ✅ `skills/` + optional-skills | ✅ 斜杠命令 + `/skillify` + Marketplace |
| Skill 触发 | 调度器 / CLI 显式 | 自动匹配 + 命名调用 | 任务匹配或按名调用 |
| **Hooks** | ✅ `config.hooks` tool_before/after | ⚠️ 插件生态 | ✅ 文件编辑、工具调用脚本 |
| **Plugins** | ❌ YAGNI | ✅ | ✅ + Marketplace 打包分发 |
| 项目规则文件 | ⚠️ `rules/` + `prompts/`（非 AGENTS.md） | ⚠️ 项目配置 | ✅ **AGENTS.md** + CLAUDE.md + Cursor 兼容 |
| 目录级 AGENTS.md | ❌ | ❌ | ✅ |
| `inspect` 式配置发现 | ✅ `geegoo inspect` + `doctor` | — | ✅ `grok inspect` 全量发现 |

## 五、工具与 MCP

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| Tool Registry | ✅ ~82 | ✅ 70+ | ✅ `xai-grok-tools` |
| Toolset 分组 | ✅ `chat_toolsets` | ✅ 28 toolset | ✅ 按场景/插件 |
| 危险操作审批 | ✅ `approval.go` + TUI y/n | ✅ approval.py | ✅ 权限提示 |
| **MCP 客户端** | ⚠️ 作 HTTP 工具消费方（GeeGooBot :3120） | ✅ | ✅ Linear/Sentry/Grafana 等 |
| **MCP 服务端** | — | — | — |
| 终端 / Shell 工具 | ❌ 刻意不做 | ✅ | ✅ 流式 bash |
| 文件读写 / 多文件编辑 | ❌ | ✅ | ✅ search-replace 重构 |
| **Code search** | ⚠️ `search_code`（信号库） | ✅ | ✅ 代码库 Grep |
| **Web search** | ✅ DuckDuckGo + 新闻聚合 | ✅ web_tools | ✅ |
| 工具 Schema 校验 | ⚠️ Meta + 空成功检测 | ✅ jsonschema | ✅ |

## 六、记忆与状态

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| 会话持久化 | ✅ SQLite + FTS5 | ✅ SQLite + FTS5 + 血缘 | ✅ |
| 跨会话 **Memory** | ✅ `recall` / `recall_yesterday_summary` | ✅ memory 插件 | ✅ 决策与上下文持久化 |
| Working memory | ✅ 工作流进度 | ✅ | ✅ |
| Evidence 可审计 | ✅ hash + evidence_refs | ❌ | — |
| 向量 / Semantic | ❌ 刻意不做 | ⚠️ 插件 | — |
| Checkpoint / Resume | ✅ workflow 按 step key | ✅ | ✅ workspace checkpoints |

## 七、工作流、调度与质检

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| 确定性工作流 Skill | ✅ pre_market / intraday / post_market | ⚠️ 偏 ReAct | — 编码计划流 |
| **Supervisor 质检** | ✅ verdict + 退避重试 | ❌ | ⚠️ Code review 行级反馈 |
| Cron / Scheduler | ✅ `geegoo scheduler` | ✅ | ⚠️ `/loop` 等 |
| Webhook 触发 | ❌ | ⚠️ | — |
| 量化验收 | ✅ `geegoo verify` | ❌ | — |
| IM 平台（飞书等） | ❌ 不做 | ✅ 20 平台 | — |
| 报告合成防失控 | ✅ evidence-only LLM + 规则锁定 result | ⚠️ 通用 LLM | — |

## 八、安全、沙箱与部署

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| **Sandbox** | ✅ 路径 + HTTP allowlist | ✅ | ✅ 隔离执行不可信代码 |
| 单二进制部署 | ✅ `geegoo` | ⚠️ Python 环境 | ✅ `grok` |
| HTTP Runtime API | ✅ `:3400` | ✅ gateway | — |
| 后台长任务监控 | ⚠️ scheduler 进程 | ✅ | ✅ Tasks / Watchers |
| 主题 / TUI 定制 | ⚠️ `display` 配置 + Hermes 风格 | ✅ | ✅ Theming |
| 轨迹训练导出 | ❌ | ✅ trajectory | — |

## 九、Git 与代码审查

| 能力 | GeeGooAgent | Hermes Agent | Grok Build |
|------|-------------|--------------|------------|
| **Git integration** | ❌ | ⚠️ 工具层 | ✅ stage/commit/push/分支 |
| **Code review** | ❌ | — | ✅ PR 前行级评审 |
| 多文件重构 | — | ⚠️ | ✅ |

## 十、差异化总结

### GeeGooAgent 独有或明显更强

1. **金融领域工具链**：盘前/盘中/盘后 Skill、GeeGoo MCP、富途/资金/策略/Bot 等 ~82 工具。
2. **Supervisor + Evidence 审计**：报告结论可追溯、recoverable 自动补跑。
3. **`geegoo verify` 量化 cutover**：字段完整率矩阵，适合生产迁移验收。
4. **工作流与 ReAct 双轨**：Skill 确定性步骤 + Chat 自由分析。

### Hermes 独有或更强

1. **多 IM 网关**（20 平台）与 **ACP**。
2. **插件生态**、更多 Provider、轨迹导出。
3. **通用工具广度**（70+），非金融场景即开即用。

### Grok Build 独有或更强

1. **编码全链路**：Plan mode、多文件编辑、Git、Code review、Headless CI。
2. **并行子 Agent + worktree** 与 **ACP** 嵌入编辑器。
3. **生态兼容**：AGENTS.md、CLAUDE.md、Cursor Skills/Hooks/MCP 即插即用。
4. **Marketplace** 分发 Skills/插件/钩子。

## 十一、GeeGooAgent 可借鉴优先级（建议）

| 优先级 | 能力 | 状态（2026-07-20） |
|--------|------|-------------------|
| P1 | `geegoo inspect` | ✅ 已上线 |
| P1 | Headless NDJSON | ⚠️ `--message --output-format ndjson`；无 `-p` 别名 |
| P2 | Plan 门控（mutating） | ✅ `plan_gate` |
| P2 | Hooks（审计） | ✅ `config.hooks` |
| P3 | 并行 `delegate_tasks` | ✅ `delegate_max_parallel` |
| P3 | AGENTS.md 加载 | ❌ 仍用 `rules/` |
| — | 文件编辑 / Git / 编码 Plan mode | **不做** |

## 十二、参考链接

| 项目 | 链接 |
|------|------|
| Grok Build | <https://github.com/xai-org/grok-build> |
| Grok 文档 | <https://docs.x.ai/build/overview> |
| Hermes 架构 | <https://hermes-agent.nousresearch.com/docs/zh-Hans/developer-guide/architecture> |
| GeeGoo Hermes 对齐 | [hermes-parity-comparison.md](../../deploy/hermes-parity-comparison.md) |
| GeeGoo 实现状态 | [implementation-status.md](../architecture/implementation-status.md) |
