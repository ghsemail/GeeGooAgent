# 分期交付与验收

> 智能体**不得跳过 Phase 0 直接写业务**。每 Phase 有明确验收；未通过不进入下一 Phase。

---

## 总览

| Phase | 目标 | 典型工期 |
|-------|------|----------|
| **0** | 平台内核空壳 | 1–2 周 |
| **1** | 首个 Workflow Skill 端到端 | 2–3 周 |
| **2** | ReAct chat + Session 持久化 | 1–2 周 |
| **3** | SkillLoader + Supervisor 自动化 | 1 周 |

---

## Phase 0 — 平台内核

### 交付物

| 层 | 交付 |
|----|------|
| L0 | EventBus, StateStore, Checkpoint |
| L1 | Gateway + mock Provider |
| L2 | Registry + mock `echo` Tool |
| L4 | ReActLoop（单轮）+ Executor |
| L5 | CLI: setup, doctor, chat（mock 回复） |
| 文档 | config.example.json, README Quick Start |

### 禁止

- 向量库 / Redis / PostgreSQL
- 任意 Shell Tool
- 真实外部 API 依赖（测试全 mock）
- 业务 Skill 逻辑

### 验收标准

1. `go test ./...` 全绿
2. `<agent> setup` 写入默认 config
3. `<agent> doctor` 配置检查通过（连通性可 skip）
4. `<agent> chat` 在 mock LLM 下完成 1 次 tool call
5. StateStore 读写 + Checkpoint 单测通过

---

## Phase 1 — 首个 Workflow Skill

### 交付物

| 层 | 交付 |
|----|------|
| L2 | 首个 Skill 全部 Tool（Catalog + Bespoke） |
| L2 | External Client + allowlist |
| L3 | Working 结构体 + Apply 全分支 |
| L4 | WorkflowRunner + Phase A/B |
| L4 | `run <skill>` + `resume --session>` |
| L5 | `skills/<first>/` 全套 + rules |
| 横切 | dry-run、write_execution_log |
| 部署 | systemd timer 或 cron 文档 |

### 实现策略（锁定）

```text
WorkflowRunner 写死或表驱动步骤顺序
  ├── 每步：Tool + Checkpoint + execution-log
  └── LLM 仅用于窄 synthesis（可选，可 stub）
```

**首个 Skill 优先 Workflow，不要先做多 Skill ReAct。**

### 验收标准

1. `--dry-run` 全流程无 mutating 副作用
2. 故意 kill 进程后 `resume` 从断点继续
3. short_circuit 路径正确（gate false 时不跑 Phase B）
4. doctor 连通性检查通过（真实 API）
5. 产出物符合 template.md（字段齐全）
6. execution-log 含真实 ISO 时间戳

---

## Phase 2 — 交互式 Agent

### 交付物

- ReAct chat 加载 Skill `tool_filter`
- Session 持久化（chatsession store）
- `max_tool_rounds` 可配置
- 可选：`agent-runtime` HTTP API
- TUI 进度（tool_start / tool_done 事件）

### 验收标准

1. chat 模式可调 Registry 中白名单 Tool
2. 多轮对话 session 可恢复
3. scheduled 模式仍不可调 mutating Tool
4. Gateway 换 model/provider 无需改 Runtime

---

## Phase 3 — 平台化

### 交付物

- `workflow/loader.go` 读 `manifest.yaml`
- `SkillLoader` 解析 mode + tool_filter
- `Supervisor.verify()` 读 `supervisor_checks.yaml`
- 新 Skill 仅加 `skills/<new>/` 不改 Go workflow 硬编码

### 验收标准

1. 新增第二个 Skill 零改 `runner.go` 业务逻辑
2. Supervisor 漏步返回 Violation 列表
3. manifest 与 Registry 不一致时 doctor 报错

---

## 成功标准模板（复制到新项目）

```markdown
## Phase 1 Done When

1. [ ] 定时触发 `<agent> run <skill>` 成功
2. [ ] 每条 work item 产出持久化 + 本地 md
3. [ ] crash 后 resume 成功
4. [ ] dry-run 与 live 步骤数一致
5. [ ] Supervisor 人工抽检 3 条通过
```

---

## 代码量参考（Go）

| Phase | 行数约 |
|-------|--------|
| 0 | ~800–1200 |
| 0+1 | ~2500–3500 |
| 0+2 | ~4000 |
| 0+3 | ~5000+ |

单文件 ≤400 行；超过则拆分。

---

## 下一步

按 [agent-build-guide.md](./agent-build-guide.md) 逐步执行；每步一个 Cursor 会话。
