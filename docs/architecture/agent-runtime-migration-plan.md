# Agent Runtime 改造计划

> **依据**：[agent-runtime-architecture.md](./agent-runtime-architecture.md)（定稿）  
> **日期**：2026-07-19  
> **范围**：Kernel / Cognition / Model Policy / Memory 接口 / 可选 Python Advisor  
> **明确不在范围**：Flutter / Web Dashboard（见文末「后续规划」）

原则：**先接口与依赖方向，再物理搬家；默认可运行路径始终全 Go；每阶段可独立合并、可回滚。**

---

## 0. 目标与非目标

### 目标

1. Kernel 只依赖 cognition **接口**，不内嵌具体策略实现细节  
2. Session SSOT 与 Memory port 分离（SQLite 仍是当前 session 实现）  
3. Model Policy 与 LLM Gateway 职责可辨  
4. 为可选 Python Advisor 留窄契约，且默认不部署  
5. 用文档 + 测试锁住 import / ownership 边界，防止 Go 泥球与「第二套 Agent」

### 非目标（本期）

- Dashboard / Flutter / 新 Web UI  
- 引入向量库或对象存储（除非独立业务需求单独立项）  
- 一夜拆完所有 `internal/` 目录改名  
- 把 loop 或 tool 执行迁出 Go  
- 为「长期 long-reasoning operator」开 Python 旁路

---

## 1. 现状基线（改造起点）

| 能力 | 现状 | 缺口（相对定稿） |
|------|------|------------------|
| ReAct loop | `internal/agent`（loop_*、tool_exec、plan_gate） | 策略与 Kernel 耦合在同包 |
| Tool / MCP | `internal/tools`、`clients/mcp` | 边界正确，保持 |
| LLM | `internal/llm.Gateway` | Policy 未独立 |
| Session | `internal/chatsession` + SQLite | 正确作 SSOT；缺与 Memory port 的显式分离文档/接口 |
| Memory | `internal/memory`、`prompt.Compressor` | 缺统一 `Recall/Store/Compress` 端口叙事与接口 |
| Cognition | 无独立包 | 需新建 |
| Python sidecar | 无 | P2 可选 |
| HTTP | `runtimeapi` | 保持；Dashboard 不驱动其改约 |

验证基线（每阶段结束应仍绿）：

```bash
go test ./internal/agent/... ./internal/runtime/... ./internal/llm/... ./internal/chatsession/... ./internal/memory/...
go build ./cmd/geegoo ./cmd/agent-runtime
```

（按仓库习惯可再跑 clarify / loop 相关 e2e。）

---

## 2. 阶段总览

| 阶段 | 主题 | 交付 | 风险 |
|------|------|------|------|
| **P0** | 定稿落地 + 边界冻结 | 本文 + 架构定稿；overview/README 索引；禁止清单写入工程约定 | 低 · **文档已完成** |
| **P1** | Cognition 接口 + Go 默认 | `internal/cognition`；Loop 改调接口；行为等价 | 中 · **代码已落地** |
| **P2** | Model Policy 抽出 | Policy 在 Gateway 之上；选模/预算可测 | 中 · **代码已落地** |
| **P3** | Memory port | 接口 + 现有实现适配；不换存储 | 中 · **代码已落地** |
| **P4** | Python Advisor（可选） | 窄 HTTP 契约 + 降级；默认关闭 | 中高 |
| **P5** | 包边界加固 | 按需物理整理、依赖检查；仍无 Dashboard | 低 |

任意阶段可暂停；**P4 非必须**，仅当有明确 ranking/evaluator 痛点再开。

---

## 3. P0 — 文档与边界冻结

### 做什么

- [x] 写入 [agent-runtime-architecture.md](./agent-runtime-architecture.md)  
- [x] 更新 [README.md](./README.md)、[../README.md](../README.md)、[overview.md](./overview.md) 入口链接  
- [x] 在 [repo-layout.md](./repo-layout.md) 增加「目标逻辑包 / 依赖方向」指引  
- [x] 在 [layers/L4-runtime/agent-loop.md](./layers/L4-runtime/agent-loop.md) 顶部加「Kernel vs Cognition」短注与链接  
- [x] 在 [layers/L1-model-gateway/README.md](./layers/L1-model-gateway/README.md) 注明未来 Policy 层  
- [ ] （可选）`docs/engineering/` 增加一条：禁止 Python 拥有 loop/tool/session 写路径  

### 完成标准

- 新人能从 architecture 索引读到定稿与本计划  
- 团队共识：Dashboard 后置；P4 默认不做  

### 不做

- 改业务代码  

**P0 文档主体已完成**；可选 engineering 约定可随 P1 一并补。  

---

## 4. P1 — Cognition 接口化（优先）

### 动机

定稿最大修正：「intelligence 不要污染 Kernel」。先接口、Go 默认，行为与今日一致。

### 做什么

1. 新建 `internal/cognition/`（或等价名，与定稿一致）：
   - `Planner` / `Ranker` / `Evaluator` 接口（可先只落地实际用到的 1～2 个）
   - Go 默认实现：把现有 `plan_gate` / LLM `tool_calls` 路径迁为默认 Planner；Ranker/Evaluator 可为 no-op
2. `internal/agent.Loop` 只依赖接口；构造时注入默认实现  
3. 单测：现有 `loop_*_test`、`plan_*_test` 全绿；增加「可替换 fake cognition」的薄测试  

### 建议顺序（降低风险）

1. 先抽 **Evaluator 或 Ranker**（更易 no-op）验证注入模式  
2. 再抽与 `plan_gate` / pending plan 相关的 **Planner 门控面**  
3. 不要一次把整段 `callLLM` 改名为 Planner 却塞进 Python  

### 完成标准

- [x] Loop 源文件委托 PlanPolicy / Evaluator / Ranker（经 `SetCognition`）  
- [x] `go test ./internal/cognition/... ./internal/agent/...` 通过；plan gate 回归保留  
- [ ] 手动 `geegoo chat` 一轮 tool call（部署前建议）  

### P1 落地摘要（2026-07-19）

- 新增 `internal/cognition`：`Ranker` / `Evaluator` / `PlanPolicy` + Go 默认实现  
- `Loop` / `Agent`：`SetCognition`；默认 `cognition.Defaults()`  
- `plan_gate` 决策与文案委托 `PlanPolicy`；回合结束调用 `Evaluator`（默认 accept）  
- `RankItems` 暴露 Ranker 钩子，供后续 recall 使用  
- PlanPolicy 决定不 hold 时，不再被 `ToolExec.planGate` 二次 skip  

### 不做

- Python  
- 大目录改名（`agent` → `agentkernel`）  

---

## 5. P2 — Model Policy 层

### 做什么

1. 在 `internal/llm`（或新建 `internal/model`）引入 **Policy / Runtime** 概念：
   - 输入：任务类型 / 复杂度提示 / session 预算  
   - 输出：model、temperature、maxTokens、是否走压缩前缀等  
2. Kernel / report synthesizer / compressor 总结调用改为经 Policy 再进 Gateway  
3. 配置：从现有 `config.json` `llm` / `agent` 段映射，保持默认行为不变  
4. 单测：简单任务 vs 复杂任务策略分支（可用 fake）  

### 完成标准

- [x] Gateway 文件职责以「provider 适配 + 重试 + 流式」为主；策略在 Policy  
- [x] 换模型策略不改 Loop 状态机  

### P2 落地摘要（2026-07-19）

- `internal/llm/policy.go`：`Policy` / `ConfigPolicy` / `ComplexityPolicy` + `CallMeta`  
- `Gateway.SetPolicy`；Chat 经 context `CallMeta` 应用 temperature / max_tokens  
- Loop chat、budget summary、report synthesis、compressor summarizer 已标 `TaskKind`  
- App 默认 `ConfigPolicy`；`ComplexityPolicy` 可选（不自动接入，避免 82 tools 抬高 max_tokens）  

### 不做

- 上一堆新 provider  
- 成本系统做满（可与现有 cost-manager 规划对齐，轻量即可）  

---

## 6. P3 — Memory port（不换存储）

### 做什么

1. 定义 Go 接口，例如：

```go
type Memory interface {
    Recall(ctx context.Context, q RecallQuery) (RecallResult, error)
    Store(ctx context.Context, rec Record) error
    Compress(ctx context.Context, in CompressInput) (CompressOutput, error)
}
```

2. 用现有 `memory` + `prompt.Compressor` + session 读路径做 **适配实现**  
3. 文档标明：Session SSOT（chatsession）≠ Memory 实现后端  
4. 为未来 episodic / semantic 留接口方法或子接口，**不实现向量库**  

### 完成标准

- [x] Chat 压缩与 recall 经 `memport.Port` / `memory.Adapter`  
- [x] SQLite / chatsession 仍为 session SSOT；无新外部依赖  

### P3 落地摘要（2026-07-19）

- `internal/memport`：`Port` + `Recall` / `Store` / `Compress` 类型与 `Noop`  
- `internal/memory.Adapter`：委托 `prompt.Compressor`、`chatsession`、`EvidenceStore`  
- Loop / SubAgent：`SetMemory`；压缩走 `mem.Compress`  
- `recall` tool：优先 `deps.Memory.Recall`（无 port 时回退原路径）  
- App：`wireChatMemory` 组装共享 `ChatMemory`  

### 不做

- 引入 Vector DB / 对象存储（单独立项）  

---

## 7. P4 — Python Advisor（可选）

**仅当**存在明确痛点（如 recall 排序、evaluator 实验）时启动。

### 做什么

1. `services/cognitive/`：无状态 HTTP 服务；输入 snapshot，输出 suggestion JSON  
2. `cognition` 下 `AdvisorClient` 实现 Ranker 或 Evaluator 之一  
3. 配置开关默认 **off**；超时 / 5xx → Go 默认  
4. 部署文档：可选 sidecar；健康检查与 Kernel 解耦  
5. 契约测试：禁止响应中带 tool_calls / 写库指令字段  

### 完成标准

- 开关 off 与无 sidecar 时行为 ≡ P3  
- 开关 on 且 sidecar 挂掉时 chat 仍成功（降级日志可观测）  

### 不做

- long-reasoning operator  
- Python 调 MCP / 写 SQLite  

---

## 8. P5 — 包边界加固

### 做什么

1. 审计 `internal/` import 图，修复违规（尤其 tools → cognition、cognition → cli）  
2. 按需物理整理：例如 `agent` 内 loop 与已迁走的策略文件清理；**不强制**改 module 路径  
3. （可选）CI 增加简单依赖检查脚本或 `go test` 内 assert  
4. 更新 [repo-layout.md](./repo-layout.md) 与定稿一致  

### 完成标准

- 定稿第 5 节依赖方向可被机器或 checklist 验证  

---

## 9. 任务拆解（执行清单）

可按 PR 粒度切割：

| ID | 阶段 | 任务 | 主要触点 | 状态 |
|----|------|------|----------|------|
| T0.1 | P0 | 索引与交叉链接 | docs/architecture/* | ✅ |
| T1.1 | P1 | 定义 cognition 接口 + 默认 | internal/cognition | ✅ |
| T1.2 | P1 | Loop 注入 Ranker/Evaluator | internal/agent | ✅ |
| T1.3 | P1 | Planner/门控委托 | plan_gate.go | ✅ |
| T1.4 | P1 | 回归测试 | *_test.go | ✅ |
| T2.1 | P2 | Policy 类型与配置映射 | internal/llm | ✅ |
| T2.2 | P2 | 调用点改经 Policy | agent、report、prompt | ✅ |
| T3.1 | P3 | Memory 接口 + 适配器 | internal/memport, memory.Adapter | ✅ |
| T3.2 | P3 | 文档：SSOT vs index | L3-memory docs | ✅ |
| T4.1 | P4 | Advisor OpenAPI/JSON 契约 | services/cognitive | |
| T4.2 | P4 | Go client + 降级 | cognition | |
| T5.1 | P5 | import 审计与文档同步 | repo-layout、CI | |

---

## 10. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 接口化时行为漂移 | 每阶段「行为等价」验收；先 no-op 注入 |
| 一次抽太多 | P1 按 Ranker → Planner 门控分 PR |
| Python 变成第二 Agent | 契约禁止 + code review checklist |
| Go 目录大搬家打断协作 | P5 才物理整理；P1–P3 以接口为主 |
| Dashboard 需求回流逼改 API | 本期冻结 UI；`runtimeapi` 只为现有客户端演进 |

---

## 11. 后续规划（本期不做）

以下单独立项，**不阻塞** P0–P5：

| 项 | 说明 |
|----|------|
| Web / Flutter Dashboard | 纯 `runtimeapi` 客户端；会话列表、任务、doctor、配置；无 agent 逻辑 |
| IDE 扩展 | 同 API；优先级可高于 Flutter |
| Vector / Object 后端 | 挂在 Memory port 下；不改变 Session SSOT |
| 多租户 / 成本账单 | 可挂 Model Policy 与 session 元数据 |

Dashboard 立项时建议产出独立短文：`docs/architecture/dashboard-client.md`（仅客户端契约），并更新本文「后续」状态。

---

## 12. 建议执行顺序（摘要）

```text
P0 文档冻结
 → P1 Cognition 接口（必做，最大架构收益）
 → P2 Model Policy
 → P3 Memory port
 → P4 Python Advisor（按需）
 → P5 包边界加固
 —— Dashboard / Flutter：另开规划 ——
```

**当前下一步**：P5 包边界加固（或按需 P4 Python Advisor）。P0–P3 已完成。
