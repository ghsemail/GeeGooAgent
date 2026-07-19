# Grok Build 功能整理

> 来源：[grok-build README](https://github.com/xai-org/grok-build/blob/main/README.md)、[docs.x.ai/build](https://docs.x.ai/build/overview)、[x.ai/cli](https://x.ai/cli)（2026-07）。
> 仓库为 SpaceXAI monorepo 定期同步的 Rust 子树，Apache 2.0 许可；**不接受外部贡献**。

## 产品定位

**Grok Build** 是 xAI 的终端 AI 编码 Agent，以全屏 TUI 为主入口，理解代码库、编辑文件、执行 Shell、联网搜索、管理长任务。同一二进制支持三种运行形态：

| 形态 | 说明 |
|------|------|
| **交互 TUI** | 全屏、可鼠标操作的主界面 |
| **Headless** | `grok -p "..."`，用于脚本与 CI/CD |
| **ACP** | Agent Client Protocol，嵌入编辑器或其他编排应用 |

底层模型默认 **grok-4.5**；亦可通过 `~/.grok/config.toml` 配置任意 OpenAI 兼容端点。

## 安装与构建

```sh
# 发布二进制
curl -fsSL https://x.ai/cli/install.sh | bash   # macOS / Linux / Git Bash
irm https://x.ai/cli/install.ps1 | iex          # Windows PowerShell

# 源码（Rust + DotSlash + protoc）
cargo run -p xai-grok-pager-bin
```

首次启动浏览器 OAuth；无头环境可设 `XAI_API_KEY`。`grok inspect` 汇总当前目录下的配置、指令、Skills、插件、Hooks、MCP 等发现结果。

## 仓库结构（Rust crates）

| 路径 | 职责 |
|------|------|
| `xai-grok-pager-bin` | 二进制入口（产物名 `xai-grok-pager`，安装为 `grok`） |
| `xai-grok-pager` | TUI：滚动区、输入、模态、渲染 |
| `xai-grok-shell` | Agent 运行时 + leader/stdio/headless |
| `xai-grok-tools` | 工具实现（终端、文件编辑、搜索等） |
| `xai-grok-workspace` | 宿主文件系统、VCS、执行、检查点 |

工具实现部分移植自 openai/codex、sst/opencode 等上游（见 `THIRD-PARTY-NOTICES`）。

## 功能清单（Everything you need to ship）

官网与 README 列出的 **19 项能力**，按工作流阶段归类如下。

### 规划与推理

| 能力 | 说明 |
|------|------|
| **Plan mode** | 复杂任务先出结构化计划；可逐步评论、改写；**批准前禁止改代码**；变更以 diff 呈现；计划文件默认 `.grok/plan.md` |
| **Deep reasoning** | 难任务分步思考（与 TUI 思考流展示配合） |
| **Q&A / ask_user** | 歧义任务多选澄清，答案流入计划；`--no-ask-user` 可关闭（含子 Agent） |

### 执行与协作

| 能力 | 说明 |
|------|------|
| **Subagents** | 并行子 Agent（宣称最多约 8 路）；`task` / `spawn_subagent` 工具；可选独立 **Git worktree**；可选 per-subagent `model` |
| **Multi-file edits** | 跨文件搜索替换式重构 |
| **Terminal execution** | 构建/测试命令，流式输出 |
| **Background tasks** | 长任务后台运行与监控（Tasks / Watchers 面板） |
| **Sandboxed execution** | 不可信代码隔离执行环境 |

### 代码库与版本控制

| 能力 | 说明 |
|------|------|
| **Code search** | 大型代码库快速 Grep / 导航 |
| **Git integration** | Stage、commit、push、分支管理 |
| **Code review** | 开 PR 前行级评审反馈 |

### 扩展与约定

| 能力 | 说明 |
|------|------|
| **Skills** | 工作流固化为可复用斜杠命令；`/skillify` 把会话捕获为新 Skill；支持 Marketplace 分发 |
| **Hooks** | 文件编辑、工具调用等事件触发脚本 |
| **MCP servers** | 连接 Linear、Sentry、Grafana 等；权限提示展示计划参数 |
| **AGENTS.md** | 按目录约定与规则；**兼容** `CLAUDE.md`、Cursor Skills、插件 |
| **Plugins / Marketplaces** | 打包 Skills、子 Agent、Hooks、MCP，一键安装或自托管 Git 源 |
| **Memory** | 跨会话持久化决策与上下文 |

### 信息与自动化

| 能力 | 说明 |
|------|------|
| **Web search** | 终端内查文档与包信息 |
| **Headless mode** | `grok -p` + `--output-format streaming-json` 等，供 CI/CD 与编排 |
| **Theming** | 颜色、字体、外观定制 |

## 配置与发现

| 层级 | 路径 | 内容 |
|------|------|------|
| 用户 | `~/.grok/config.toml`（Windows：`%USERPROFILE%\.grok\config.toml`） | 默认模型、自定义 `[model.*]`、`[models].default` |
| 项目 | `.grok/config.toml` | 项目级覆盖 |
| 计划 | `.grok/plan.md` | Plan mode 默认计划文件 |

`grok inspect` 列出：配置来源、instructions、skills 路径、bundled 扩展、MCP/插件/钩子来源。

## 与 Claude Code / Cursor 生态的兼容策略

Grok Build 明确以 **低迁移成本** 为卖点：

- 直接读取 `AGENTS.md`、`CLAUDE.md`
- 可加载 Cursor / Claude 的 Skills、Hooks、MCP、插件配置
- ACP 对标 IDE 集成场景

## 非目标 / 限制（开源树）

- 根 `Cargo.toml` 为生成文件，只改各 crate 的 `Cargo.toml`
- Windows 构建为 best-effort，CI 未覆盖
- 外部 PR 不接受（见 `CONTRIBUTING.md`）
- 订阅：早期 Beta 面向 SuperGrok / X Premium Plus（产品策略，与开源代码树分离）

## 文档索引（上游）

| 主题 | 位置 |
|------|------|
| 用户指南 | 仓库内 `crates/codegen/xai-grok-pager/docs/user-guide/` |
| 在线文档 | <https://docs.x.ai/build/overview> |
| 变更日志 | <https://x.ai/build/changelog> |
| 认证 | `user-guide/02-authentication.md` |

## 对 GeeGooAgent 的启示（摘要）

Grok Build 与 GeeGooAgent **场景不同**：前者是通用编码 CLI，后者是金融工作流 Agent。仍可借鉴的共性能力：

1. **Plan mode + 批准门控** — 复杂金融工作流（如 Bot 创建）可先出计划再执行。
2. **Headless / JSON 流** — `geegoo run` / scheduler 与 CI 验收可统一非交互输出格式。
3. **inspect 式自检** — 类似 `geegoo doctor`，一次性展示 rules、skills、MCP、toolset 发现结果。
4. **并行子 Agent + worktree** — 多标的并行分析时可参考隔离模型（GeeGoo 已有 `delegate_task`，无 worktree）。
5. **Hooks** — 文件/工具事件钩子可用于审计或合规流水线。

完整对比见 [comparison.md](./comparison.md)。
