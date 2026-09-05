# 工具平台彻底改造方案（Claude Code + Codex 对齐）

> 状态：**M1–M5 已落地**（2026-08-31）；默认开启 `policy_v2`、`defer_load_tools`、`tool_fragment_inject`  
> 目标：将 GeeGooAgent L2 工具层做成「Claude Code 单工具生命周期 + Codex 声明式 policy / fragment 管道」的 Go 版同构实现。

## 1. 背景与动机

### 1.1 现状（M1，已落地）

`internal/tools/Tool` 已扩展 `ToolSpec`，在 `Register()` 时解析：

- `Policy`（allow / prompt / forbidden）
- `ReadOnly`、`ConcurrencySafe`
- `MaxResultChars`、`Timeout`
- `DeferLoad`、`UserFacingLabel`

消费方：`approval.go`、`timeouts.go`、`metadata.go`、`tool_exec.go`、`loop_plan.go`、`inspect`、`verify`。

详见 [tool-spec.md](./tool-spec.md)。

### 1.2 仍存在的差距

| 能力 | Claude Code | Codex | GeeGooAgent（M1 后） |
|------|-------------|-------|----------------------|
| 权限模型 | `checkPermissions` | execpolicy | `tool_policy.yaml` ✅ 基础版 |
| 只读 / 并发 | `isReadOnly` / `isConcurrencySafe` | 隐式 | ✅ 推断 + 可覆盖 |
| 结果上限 | `maxResultSizeChars` | fragment 压缩 | ✅ 统一 JSON 截断 |
| **每工具渲染** | `renderResultForAssistant` | fragment | ❌ 无 per-tool RenderFunc |
| **延迟暴露** | `deferLoading` + ToolSearch | toolset | ⚠️ 仅元数据，未接运行时 |
| **Fragment 注入** | — | context fragments | ⚠️ recall 已接，tool 结果未接 |
| Schema 强类型 | Zod | JSON Schema | ⚠️ `map[string]any` |
| 金融 plan gate | — | — | ✅ GeeGoo 特有，保留 |

**对齐度估计：** M1 ≈ 70%；终态目标 ≈ 90～95%。

### 1.3 改造原则

1. **SSOT**：每个工具一份 `ToolDefinition`，Registry / Loop / MCP / 运营台共用。
2. **可灰度**：每阶段 feature flag，可独立上线与回滚。
3. **金融安全**：`plan_gate` + `prompt`/`forbidden` 不削弱，只加强。
4. **不复制闭源 UX**：对齐架构，不对齐 Claude 桌面弹窗形态。

---

## 2. 终态架构

```text
┌─────────────────────────────────────────────────────────────┐
│  Schema 暴露层（Claude deferLoading + ToolSearch）           │
│  Core Toolset (~15) │ DeferLoad 池 │ discover / activate    │
└───────────────────────────┬─────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Policy 决策层（Codex execpolicy）                            │
│  tool_policy.yaml → engine → forbidden > prompt > allow      │
│  + plan_gate（GeeGoo 金融确认）                              │
└───────────────────────────┬─────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Execute 执行层（Claude call）                                │
│  Validate → Permission → Handler → Timeout → Hooks           │
│  → RenderForAssistant                                        │
└───────────────────────────┬─────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Fragment 注入层（Codex context）                             │
│  Composer: ToolResult / Recall / Hook / WorkingState         │
└───────────────────────────┬─────────────────────────────────┘
                            ▼
                         LLM 上下文
```

### 2.1 终态核心类型

```go
// internal/tools/definition.go（规划）

type ToolDefinition struct {
    Name        string
    Description string
    Parameters  SchemaSource   // JSON Schema | struct 生成 | catalog HTTPSpec

    Execute     ToolHandler    // 对标 Claude call()

    // Claude 对齐
    ReadOnly         bool
    ConcurrencySafe  bool
    MaxResultChars   int
    DeferLoad        bool
    RenderForLLM     RenderFunc   // 对标 renderResultForAssistant
    UserFacingLabel  string

    // Codex 对齐
    Policy           PolicyAction // allow | prompt | forbidden
    ScopeEmit        []string     // 调用后写入 active_scopes（可选）

    // GeeGoo 金融域
    Domain           ToolDomain
    Toolsets         []string
    RequiresMCP      bool
    WorkflowOnly     bool
}
```

### 2.2 与 Claude Code `Tool` 接口对照

| Claude Code | GeeGoo 终态 |
|-------------|-------------|
| `name` | `Name` |
| `description` | `Description` |
| `inputSchema` (Zod) | `Parameters` / `SchemaSource` |
| `call` | `Execute` |
| `checkPermissions` | `Policy` + `policy.Engine` + `plan_gate` |
| `isReadOnly` | `ReadOnly` |
| `isConcurrencySafe` | `ConcurrencySafe` |
| `maxResultSizeChars` | `MaxResultChars` + `RenderForLLM` |
| `deferLoading` | `DeferLoad` + `discover_tools` |
| `userFacingName` | `UserFacingLabel` |
| `renderResultForAssistant` | `RenderForLLM` |

### 2.3 与 Codex 对照

| Codex | GeeGoo 终态 |
|-------|-------------|
| `execpolicy` prefix rules | `tool_policy.yaml` + `policy/engine.go` |
| `context` fragments | `internal/context` Composer + 全来源注入 |
| `tool_result` item_type | 已有 SSE/NDJSON；fragment 接管内容 |
| `hook_additional_context` | `HookInjectFragment`（P2-4） |
| MCP tools 暴露 | `geegoo mcp serve`（已有 MVP） |

---

## 3. 分阶段实施计划

建议 **4 个阶段、约 3～4 周**（1 人全职）。每阶段独立可部署、`geegoo verify agent-loop` 回归。

### 阶段 A：工具定义 SSOT（3～4 天）

**目标：** 消灭 `domains.go` / `metadata.go` 重复推断；catalog 与 bespoke 同源。

| 任务 | 产出 |
|------|------|
| `ToolDefinition` 替换 `Tool` + `ToolSpec` | `internal/tools/definition.go` |
| `catalog.AllHTTP()` → `DefinitionFromHTTPSpec()` | 61 个 HTTP 工具批量生成 |
| `bespoke.go` 显式声明 Spec 覆盖 | 21 个 bespoke 可自定义 render |
| `domains.go` 改为读 `Definition.Toolsets` | 不再靠名字反查 |
| Registry API 统一 | `Definitions()` / `Schemas(opts)` / `Execute()` |

**验收：**

- `geegoo inspect` 展示各 domain 的 policy / readonly / defer 分布
- 注册工具数不变；`bootstrap_test` 通过

**Feature flag：** 无（内部重构，行为不变）

---

### 阶段 B：Policy 引擎 v2 + Schema 过滤（3～4 天）

**目标：** 对标 Codex execpolicy；`forbidden` 工具对 LLM 不可见。

| 任务 | 产出 |
|------|------|
| `internal/tools/policy/engine.go` | `match` / `not_match` / 优先级 |
| 决策链 | `forbidden > prompt > allow`，与 plan_gate、workflow `Approved` 合并 |
| `Schemas(ExcludeForbidden: true)` | chat 默认不暴露 `delete_*` 等 |
| 生产默认 policy 模板 | 见 [tool_policy.example.yaml](./tool_policy.example.yaml) |
| `geegoo inspect --policy` | 规则数 + 样例命中 |

**验收：**

- `verify` 卡片：`forbidden tool absent from schema`
- `delete_*` 不在 chat schema；强行调用返回明确错误

**Feature flag：** `config.agent.policy_v2`（默认 false → 灰度 → true）

**与 Claude 差异（刻意保留）：** 金融写操作用 plan_gate + 运营台确认，不照搬 Claude 桌面权限弹窗。

---

### 阶段 C：延迟加载 + ToolSearch（4～5 天）

**目标：** 对标 Claude `deferLoading`；降低 80+ schema 噪音与误调用。

| 任务 | 产出 |
|------|------|
| **Core toolset**（~15）常驻 schema | search_code, recall, clarify, get_current_price, delegate_*, … |
| `DeferLoad=true` 默认不进 schema | 接上 `Schemas` 过滤 |
| Meta tool：`discover_tools(query)` | FTS / 关键词搜 name+description+domain |
| Meta tool：`activate_toolset(id)` | 会话级临时展开 bot_manager 等 |
| Session `active_toolsets` | Loop 每轮 merge schema |
| 并入 `SkillToolExpander` | skill 匹配 = 自动 activate |

**验收：**

- 默认 chat schema ≤ 20 个工具
- 用户「帮我建 DCA bot」→ discover → activate → 出现 `create_dca_bot`
- 首轮 system+tools 字节数下降 ≥ 40%（对比 M1）

**Feature flag：** `config.agent.defer_load_tools`（默认 false）

**可选 C+（语义 ToolSearch）：** 对 `CatalogItem` 建 embedding 索引；不阻塞 C 上线。

---

### 阶段 D：Fragment 管道接管全部注入（4～5 天）

**目标：** 对标 Codex context fragment；tool / hook 输出统一走 Composer。

| 任务 | 产出 |
|------|------|
| `appendToolResults` → `ToolResultFragment` | 内容经 Composer 预算控制 |
| `RenderForLLM` 分级 | Tier0 默认 JSON；Tier1 list 摘要；Tier2 大 payload 自定义 |
| Hook stdout → `HookInjectFragment` | 完成 roadmap P2-4 |
| Compressor 顺序 | 先 drop 低优先级 fragment，再 truncate message |
| 事件 | `context_fragment_applied` 含 `source=tool|hook|recall` |

**设计决策（默认）：**

| 问题 | 决策 | 理由 |
|------|------|------|
| OpenAI tool message 格式 | **保留** `role=tool` + `tool_call_id` | 网关 function calling 兼容 |
| 全量 Data | 存 `StepRecord.Meta` + 可选 episodic | LLM 只见摘要，运营台见全量 |

**验收：**

- 长 list 工具后 prompt tokens 可控
- Loop trace 有 fragment 事件且可区分 source

**Feature flag：** `config.agent.tool_fragment_inject`（默认 false）

**依赖：** P1-1 Context Fragments（已落地）

---

### 阶段 E：Schema 强类型 + 生命周期（5～7 天，可选加深）

**目标：** 尽量靠近 Claude Zod `inputSchema` 的开发体验与安全性。

| 任务 | 产出 |
|------|------|
| `internal/tools/schema/` | struct tag → JSON Schema 生成 |
| 21 个 bespoke 改 struct 参数 | 编译期 + 运行时双校验 |
| catalog OpenAPI 自动生成 Parameters | 消除手写 `map[string]any` |
| 生命周期钩子 | `BeforeCall` / `AfterCall` / `OnError` |
| `geegoo doctor --tools` | schema 漂移检测 |

**验收：**

- bespoke 工具参数错误在 `Validate` 阶段 100% 拦截
- doctor 无 catalog ↔ registry 漂移

**Feature flag：** 无（增量替换 Parameters 来源）

---

## 4. 里程碑与对齐度

| 里程碑 | 内容 | 验收 | 对齐度 |
|--------|------|------|--------|
| **M1** | ToolSpec 元数据（已完成） | `verify agent-loop` 18/18 | ~70% |
| **M2** | 阶段 A + B（ToolDefinition、Policy v2、SchemaOptions） | verify + inspect policy | ~80% |
| **M3** | 阶段 C（Core toolset、discover/activate） | 默认 schema ≤20 tools | ~85% |
| **M4** | 阶段 D（Fragment 管道 + Hook inject） | fragment 覆盖 tool+hook | ~90% |
| **M5** | 阶段 E（catalog drift、`doctor --tools`） | `geegoo doctor --tools` | ~95% |

推荐时间线：

```text
Week 1:  A + B   → 可部署，policy 更严、metadata 统一
Week 2:  C       → 可部署，schema 瘦身
Week 3:  D       → 可部署，token / audit 达标
Week 4:  E 按需  → 工程质量上限
```

---

## 5. 明确做不到 / 不应做的部分

| 项 | 理由 |
|----|------|
| 1:1 复制 Claude Code 桌面权限 UI | 产品形态为 Flutter 运营台 + plan_gate；应做强 audit，不对齐弹窗 |
| TS / Zod 同款 DX | Go 用 struct + codegen 达到同等**安全性**，非同等写法 |
| 82 工具全部手写 `RenderForLLM` | 不经济；Tier0 默认 + ~15 高频 Tier1/Tier2 覆盖 90% token 问题 |
| 复制 Claude 专有 ToolSearch 模型 | 闭源；用 FTS / embedding 近似，效果可验收 |
| 去掉 plan_gate | 金融写操作必须保留；GeeGoo 优势 |
| 第三方 Plugin ABI（动态 .so） | roadmap 已定走 MCP Server，更安全 |
| 一周内完成阶段 E | 21 bespoke struct 化 + OpenAPI codegen 宜与 A～D 并行，不阻塞上线 |

---

## 6. 风险与回滚

| 风险 | 缓解 |
|------|------|
| discover 找不到工具 | Core set 保留 search_code / recall；失败 fallback 全量 toolset |
| forbidden 误伤 workflow | workflow 使用 `Schemas(IncludeForbidden: true)` + `Approved: true` |
| fragment 破坏 tool_call 配对 | 保留 assistant `tool_calls` + tool role；只改 content 来源 |
| 运营台依赖旧 SSE flat 字段 | 继续双写 flat + 嵌套 `data` |
| 灰度期间行为不一致 | 每阶段独立 feature flag；关 flag 回退上一里程碑 |

建议配置项（`config.json` 规划）：

```json
{
  "agent": {
    "policy_v2": false,
    "defer_load_tools": false,
    "tool_fragment_inject": false
  }
}
```

---

## 7. 涉及文件索引

| 阶段 | 主要路径 |
|------|----------|
| M1（已有） | `internal/tools/spec.go`, `policy.go`, `render.go`, `registry.go` |
| A | `internal/tools/definition.go`, `catalog/`, `bespoke.go`, `domains.go` |
| B | `internal/tools/policy/engine.go`, `internal/config/config.go` |
| C | `internal/tools/discover.go`, `internal/agent/loop.go`, `internal/chatsession/` |
| D | `internal/agent/loop_plan.go`, `internal/context/`, `internal/tools/hooks.go` |
| E | `internal/tools/schema/`, `cmd/geegoo/doctor.go` |

相关文档：

- [tool-spec.md](./tool-spec.md) — M1 已实施说明
- [tool_policy.example.yaml](./tool_policy.example.yaml) — 策略配置示例
- [L2 ToolRegistry](../architecture/layers/L2-tools/registry.md)
- [Codex 对标路线图](../benchmark/agent-loop/codex-inspired-roadmap.md) — P2-3 / P2-4

---

## 8. 验收命令汇总

```bash
# 单元 + 集成
go test ./internal/tools/... ./internal/agent/...

# Loop 离线卡片（随阶段扩展）
go run ./cmd/geegoo verify agent-loop --offline

# 运行态快照
go run ./cmd/geegoo inspect

# 阶段 B 后
go run ./cmd/geegoo inspect --policy

# 阶段 E 后
go run ./cmd/geegoo doctor --tools
```

---

## 9. 变更记录

| 日期 | 版本 | 说明 |
|------|------|------|
| 2026-08-31 | M1 | ToolSpec 元数据层落地，见 [tool-spec.md](./tool-spec.md) |
| 2026-08-31 | 规划 v1 | 本文档：阶段 A～E 彻底改造方案 |
