# 智能体分步构建指南

> **用途**：下一个 Cursor / 自托管智能体读取本文件后，按 Step 0→15 顺序生成完整 Agent。  
> **前提**：已读 [README.md](./README.md)、[layers.md](./layers.md)、[repo-layout.md](./repo-layout.md)。

---

## 1. 执行铁律

| 铁律 | 说明 |
|------|------|
| **一步一会话** | 每 Step 新建 Agent 会话，避免上下文污染 |
| **先读后写** | 每步 `@` 引用「必读文件」，禁止凭记忆编 API |
| **先测后并** | 本 Step 测试未绿，禁止下一步 |
| **小 diff** | 每步 ~300–500 行；过大则拆 Step |
| **dry-run 优先** | Phase 1 完成前默认 `dry_run=true` |
| **写清不做什么** | 每步 Prompt 末尾列出「本步禁止」 |

**禁止**：一个会话「帮我把整个 Agent 写完」。

---

## 2. 项目初始化 Prompt（Step 0 之前）

复制到新会话，替换 `<AgentName>`、`<agent>`、领域描述：

```markdown
你要从零创建一个自托管 Agent 项目 `<AgentName>`。

## 权威蓝图（必须遵守）
@docs/architecture/platform-blueprint/README.md
@docs/architecture/platform-blueprint/repo-layout.md
@docs/architecture/platform-blueprint/layers.md
@docs/architecture/platform-blueprint/phases.md

## 领域
- 用途：<一句话，如「客服工单自动回复 Agent」>
- 外部 API：<base_url 说明>
- 首个 Skill 名：<first_skill>
- 语言：Go 1.22+

## 本步只做
- 创建 repo-layout 中的目录树空壳
- go mod init、config.example.json、README Quick Start
- 禁止实现业务 Tool

## 验收
- `go build ./cmd/<agent>` 通过（main 可只 print usage）
```

---

## 3. Step 0–15 指令模板

### Step 0 — 脚手架

```markdown
@docs/architecture/platform-blueprint/repo-layout.md
@docs/architecture/platform-blueprint/layers.md §L0

实现 Phase 0 脚手架：
1. cmd/<agent>/main.go — 子命令 stub：setup, doctor, chat, run, resume
2. internal/config — Load(config.json), DefaultPath ~/.<agent>/
3. config.example.json

禁止：业务 Tool、真实 HTTP、LLM。

验收：go build ./cmd/<agent>；setup 写 config；doctor 检查文件存在。
测试：config 加载单测。
```

### Step 1 — L0 四件套

```markdown
@docs/architecture/platform-blueprint/layers.md §L0

实现 internal/infra：
1. EventBus — Emit(event, payload)，内存同步
2. StateStore — Save/Load JSON 文件到 output_dir
3. CheckpointManager — Save/Load Checkpoint 结构

禁止：import runtime/tools/llm。

验收：TestStateStoreRoundTrip、TestCheckpointSaveLoad 通过。
```

### Step 2 — L1 Gateway

```markdown
@docs/architecture/platform-blueprint/layers.md §L1

实现 internal/llm：
1. types.go — Message, ToolSchema, Response, ToolCall
2. Provider 接口 + mock.go（固定返回 1 个 tool_call）
3. gateway.go — Chat 带 3 次重试

禁止：Runtime 包 import openai 直连。

验收：mock Provider 单测；Gateway 重试单测。
```

### Step 3 — L2 Registry + mock Tool

```markdown
@docs/architecture/platform-blueprint/layers.md §L2

实现 internal/tools：
1. registry.go — Register, Execute, Schemas（与 GeeGooAgent 同签名）
2. bootstrap.go — RegisterAll 占位
3. mock echo Tool：name=echo, 返回 args 回显

ToolContext 含：SessionID, UserToken, DryRun, Step, EventBus, StateStore

验收：TestRegistryExecuteEcho；Schemas 含 echo。
```

### Step 4 — L4 ReAct 单轮

```markdown
@docs/architecture/platform-blueprint/layers.md §L4

实现 internal/runtime：
1. session.go — AppendMessage, LLMMessages
2. executor.go — 委托 Registry
3. react.go — RunTurn：user → gateway → 1 tool → assistant

max_tool_rounds 默认 8。tool result JSON ≤6000 字符。

验收：TestReActSingleToolRound（mock Gateway + echo Tool）。
```

### Step 5 — internal/app 组装 + chat CLI

```markdown
@docs/architecture/platform-blueprint/repo-layout.md

实现 internal/app/app.go — LoadFromConfigPath 组装 infra+registry+gateway+react。
实现 cmd/<agent>/chat.go — 读 stdin 一行，RunTurn，打印回复。
实现 cmd/<agent>/ops.go — setup, doctor（配置项检查，skip 连通性）。

验收：chat 在 mock 下 echo 成功；doctor 缺 token 时 FAIL 提示清晰。
```

### Step 6 — External Client 骨架

```markdown
@docs/architecture/platform-blueprint/layers.md §L2

实现 internal/clients/<domain>/client.go：
1. Post(path, body) — Bearer api_key
2. HTTP allowlist（config sandbox.allowed_hosts）
3. 超时 15s

禁止：在 runtime 里直接 http.Post。

验收：httptest.Server mock；allowlist 拒绝非法 host 单测。
```

### Step 7 — L3 Working Memory

```markdown
@docs/architecture/platform-blueprint/layers.md §L3

为首个 Skill 定义 Working 结构体（internal/memory/models.go）：
- session_id, skill, phase, gate_ok, items map, steps_completed[]

实现 WorkingStore Create/Load/Save/Apply：
- 每个 Step 7 已知 Tool 名一个 Apply 分支

验收：Apply 单测覆盖每个 Tool 分支。
```

### Step 8 — L4 WorkflowRunner

```markdown
@docs/architecture/platform-blueprint/layers.md §L4 §4.3

实现 internal/workflow/runner.go：
- Step, Run, RunFrom(resume)
- 每步 checkpoint.save
- short_circuit 支持

internal/workflow/steps.go — 硬编码首个 Skill Phase A/B（后续 Step 12 对齐 manifest）

验收：TestWorkflowRunnerDryRun 3 步；TestResumeSkipsCompletedSteps。
```

### Step 9 — run / resume CLI

```markdown
@docs/architecture/platform-blueprint/repo-layout.md §CLI

实现 cmd/<agent>/run.go — run <skill> --dry-run
实现 resume —resume <session_id>

App.RunFirstSkill() wiring。

验收：dry-run 全流程 completed；kill 后 resume 成功（单测模拟 completedStep）。
```

### Step 10 — Tool Catalog + 首批 HTTP Tools

```markdown
@docs/architecture/platform-blueprint/layers.md §L2 Catalog

实现 internal/tools/catalog/catalog.go — 声明式 HTTP Tool 列表。
RegisterHTTPFromCatalog — 自动注入 user_token、dry_run 短路。

注册首个 Skill prelude 所需 Tools（check_gate, list_work_items 等）。

验收：catalog contract test；NeedsUserToken 缺 token 返回 error。
```

### Step 11 — Bespoke Tools + Apply 补全

```markdown
@docs/architecture/platform-blueprint/skill-pack.md

实现 bespoke.go：write_execution_log, save_local_*, synthesize_*（可 stub）
补全 Working.Apply 全部分支。

验收：Phase A+B 全 Tool 在 dry-run 下跑通。
```

### Step 12 — skills/ 资产 + rules

```markdown
@docs/architecture/platform-blueprint/skill-pack.md

创建 skills/<first_skill>/：
- SKILL.md, manifest.yaml, workflow.md, template.md, supervisor_checks.yaml

创建 rules/ 至少 2 个 md。

workflow/steps.go 与 manifest.yaml 对齐（步骤名一致）。

验收：manifest 中每个 tool 在 Registry.ListNames 中存在。
```

### Step 13 — doctor 连通性 + 真实 API 冒烟

```markdown
@docs/architecture/platform-blueprint/phases.md §Phase 1

实现 doctor/connectivity.go —  ping base_url 1 个只读端点。

禁止：在单测中访问真实 API。

验收：本地 config 配好后 doctor 全绿；--dry-run run 仍全绿。
```

### Step 14 — LLM synthesis（可选窄任务）

```markdown
@docs/architecture/platform-blueprint/layers.md §L1

在 Bespoke Tool synthesize_* 内调 Gateway，输入 Working 摘要，输出 template 字段。
structured output 或 markdown 解析。

禁止：用 LLM 编排逐步 API 调用（那是 ReAct 的事）。

验收：mock LLM 返回固定 markdown；Apply 写入 working。
```

### Step 15 — deploy + Supervisor stub

```markdown
@docs/architecture/platform-blueprint/phases.md
@docs/architecture/platform-blueprint/skill-pack.md §supervisor

1. deploy/systemd/<agent>-<skill>.timer — 调 run <skill>
2. internal/supervisor/verify.go — 读 supervisor_checks.yaml（先实现 2 条 assert）

验收：run 结束后 supervisor 无 violation；timer unit 语法正确。
```

---

## 4. Cursor Rules 模板

写入新仓库 `.cursor/rules/agent-platform.md`：

```markdown
- 实现依据：docs/architecture/platform-blueprint/
- 禁止：LangChain、向量库、Shell Tool、Runtime 直连 HTTP/LLM
- Phase 1：WorkflowRunner 优先，ReAct 仅 chat
- 每步必须测试；HTTP 用 httptest mock
- 单文件 ≤400 行
- 只改本 Step 列出的文件
- 交付末尾：测试文件列表、go test 结果
```

---

## 5. 领域映射工作表（新 Agent 必填）

智能体在 Step 0 前填写：

| 蓝图概念 | 你的 Agent |
|----------|------------|
| `check_gate` | |
| `list_work_items` | |
| `fetch_context` | |
| `persist_result` | |
| Phase B 迭代字段 | |
| `user_token` 来源 | |
| `base_url` | |
| systemd 触发时间 | |

GeeGoo 映射示例见 [../overview.md](../overview.md)。

---

## 6. 常见返工原因

| 症状 | 根因 | 修复 |
|------|------|------|
| LLM 漏调 API | 用 ReAct 跑批量 | 改 WorkflowRunner |
| resume 重复写 | 无 checkpoint | 每步 save step_index |
| prompt 爆 token | 整段 API JSON 进 messages | Working.Apply + Summary |
| 定时任务删数据 | scheduled 未过滤 Tool | manifest mode + filter |
| 测试 flaky | 依赖真实 API | httptest + mock Gateway |

---

## 7. 完成后清单

- [ ] `go test ./...` 全绿
- [ ] README：setup / doctor / run / chat 四命令
- [ ] config.example.json 无真实密钥
- [ ] skills/<first>/manifest.yaml 与 Registry 一致
- [ ] `--dry-run` 与 live 步骤一致
- [ ] systemd timer 或 cron 文档
- [ ] 架构 README 链接到 platform-blueprint

---

## 8. GeeGooAgent 对照源码

| 蓝图模块 | 参考文件 |
|----------|----------|
| Registry | `internal/tools/registry.go` |
| ReAct | `internal/runtime/react.go` |
| Workflow | `internal/workflow/runner.go` |
| Working Apply | `internal/memory/working.go` |
| App wiring | `internal/app/app.go` |
| manifest 示例 | `skills/pre_market/manifest.yaml` |

实现 GeeGoo 域：叠加 [engineering/requirements.md](../../engineering/requirements.md) 与 [domains/](../domains/)。
