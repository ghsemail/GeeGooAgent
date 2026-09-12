# Eval 机制构建指南

> **用途**：给下一个 Cursor / 自托管智能体阅读。按本文机制落地一套 **Agent Eval**，不要复制任何业务领域内容。  
> **前提**：Agent 已有 Session、ReAct Loop、Intent Planner（或等价路由）、Tool Registry。  
> **原则**：本文只讲 **评测机制**。领域话术、实体名、业务工具名一律由目标 Agent 自行替换。

---

## 0. 先读这段：Eval 是什么

Eval 不是单元测试，也不是人工点几条对话。它是一套 **可重复、可落库、可批量跑** 的验收系统，用来回答三件事：

1. **路由对不对**：这一轮用户输入，Planner 给出的 `domain / mode / act` 是否符合预期。
2. **执行对不对**：Loop 有没有调用该调用的工具、有没有误触发禁止工具。
3. **回复对不对**：助手文本是否覆盖验收点（结构化关键词 + LLM 语义裁判）。

三件事必须拆开检查。任何一项失败，整条用例失败。不要把「看起来聊得通」当成通过。

```text
Case (脚本 + 期望)
        │
        ├─ Plan-only ──► Planner.Plan(user_text, last_domain) ──► 路由断言
        │
        └─ Live ──► 独立 Session ──► 按脚本发用户轮次 ──► 写 Session Trace
                         │
                         └─ Verify ──► intent ∧ execution ∧ reply ∧ judge
```

---

## 1. 总体构建方法（先机制，后用例）

按这个顺序搭，不要先堆话术。

| 顺序 | 层 | 做什么 | 不做 |
|------|----|--------|------|
| 1 | **观测点** | Loop 每轮把路由快照、工具调用写入 Session Metadata | 不要从 prompt 文本里反推路由 |
| 2 | **用例模型** | 定义 Case / Dialogue / Expect* / Judge | 不要把期望写死在 UI |
| 3 | **双入口** | Plan-only（秒级）+ Live（分钟级） | 不要用 Live 测纯分类 |
| 4 | **Verify 管道** | 结构化检查在前，LLM judge 在后 | 不要只靠 judge |
| 5 | **代码即源** | Go/源码定义用例 → 生成 SQL + JSON | 不要只在 DB 里手改 |
| 6 | **自动化 Job** | Catalog → Suite → Job → Item | 不要并行交错跑 Live |
| 7 | **运营面** | HTTP API + 一个自动测评页 | 不要让评测只活在本地脚本 |

领域用例是第 8 步：用 **分类 × 对话形态** 填表，而不是按业务名词堆 case。

---

## 2. 必须先有的运行时观测点

没有这些字段，Eval 无法做结构化校验。Loop **每完成一轮用户输入** 必须写入 Session Metadata：

| Metadata key | 内容 | 用途 |
|--------------|------|------|
| `last_turn_plan` | 本轮 `{domain, mode, act, sop, tools_allow}` | intent 检查（默认看最后一轮） |
| `turn_plan_trace` | 每轮一条路由快照（1-based） | 澄清脚本要核 **首轮** 路由 |
| `last_turn_tools_called` | 本轮实际调用的工具名列表 | 兼容旧校验 |
| `turn_tools_trace` | 每轮一条 `{turn, tools[]}` | 区分「评判轮」与「整段 session」 |

派生概念（Verify 时计算，不必另存）：

- **judged tools**：`turn_tools_trace` 最后一轮的工具（评判轮）。
- **session tools**：整段 session 工具并集。
- **dialogue snapshot**：过滤后的 user/assistant 文本，给 UI 和 judge。

写入时机：`Agent.Run` 成功结束、把 runtime session 同步回 chat session 之后。缺快照 = 该轮未完成，Verify 直接 fail。

```text
用户轮 N
  → Planner 产出 domain/mode/act
  → Loop 调工具 / 生成回复
  → SyncLastTurnPlan(...)
  → AppendTurnToolsTrace(toolNames)
  → Save(session)
```

---

## 3. 两种评测模式（必须同时有）

| | Plan-only | Live |
|--|-----------|------|
| **测什么** | IntentPlanner 分类 | 真实 Loop：路由 + 工具 + 回复 |
| **入口** | `POST .../eval/run-turn-plan` | 先跑对话，再 `POST .../cases/{id}/verify` |
| **耗时** | 秒级 | 分钟级（单轮可到数分钟） |
| **Session** | 不占 | **每条用例独立 session** |
| **上下文** | 用 `last_domain` 字段模拟上一轮 | 用真实 `setup` 轮次堆上下文 |
| **能不能 `cases/{id}/run`** | 可以（内部转套件） | **禁止**；直接 run 应 400 |

Live 的硬规则：

- 禁止用「假 session / 注入假回复」冒充 Live。
- 禁止把多条用例塞进同一个 session。
- `session_cleanup: before_run`：开跑前清空对话，只清 session，不清历史 eval 日志。

Plan-only 套件用 `LastDomain` 表示「上一轮已经停在某 domain」，用来测粘性路由与切换，不发真实前置消息。

---

## 4. 分层与源码职责

把 Eval 拆成四个包/模块，禁止揉进 Loop 里。

```text
internal/eval/                 # 纯逻辑：用例、Normalize、Verify、Judge 接口
internal/domaincatalog/        # ExecutionProfile 契约（领域可替换，机制通用）
internal/runtimeapi/           # HTTP：cases / verify / catalog / jobs
internal/infra/*schema*        # 表 DDL + 由生成器写入的 seed
scripts/eval/                  # 生成 JSON、同步 DB、远程冒烟
ops UI                         # 自动测评 Tab / HTML 页，只调 API
```

| 模块 | 允许依赖 | 禁止 |
|------|----------|------|
| `eval` | session 读接口、Planner 接口、LLM Provider | HTTP、DB、具体业务 Tool 实现 |
| `domaincatalog` 的 profile | 工具名字符串 | 调用真实工具 |
| `runtimeapi` | eval + session store + Agent.Run | 在 handler 里写校验规则 |
| 生成器 | 只读 `eval` 导出的用例函数 | 手写 SQL 行 |

**代码是用例的唯一源。** DB 只是运行时副本。改用例 = 改源码 → 跑生成器 → 迁移 DB。

---

## 5. 用例功能构建逻辑

### 5.1 一张用例由什么组成

Dashboard / DB 一行 `agent_eval_cases`：

| 列 | 作用 |
|----|------|
| `id` | 稳定主键，建议 `{suite}_{turn_id}` |
| `title` / `description` / `steps_json` | 给人看的步骤说明 |
| `options_json` | **全部行为配置**（脚本、期望、judge） |
| `sort_order` / `enabled` | 列表顺序与开关 |
| `user_id` | `''` = 全局 seed；非空 = 用户私有 |

`options_json` 的逻辑结构（实现时用强类型，JSON 只是落库形态）：

```text
TurnPlanCaseOptions
├── 运行控制
│     category, plan_only, session_cleanup, dual_model_eval
├── 脚本
│     dialogue[]          # 权威
│     setup_messages[]    # 遗留：dialogue 之前的用户轮
│     message             # 遗留：最后一轮用户话
│     clarify_reply       # 澄清默认回复
├── 期望
│     expect_intent       # domain / mode / sop / act
│     expect_execution    # profile 或 legacy_require_tools + forbid_tools
│     expect_reply        # rubric / must_cover / must_not
│     judge               # enabled / model_slot / min_score
└── 兼容扁平字段
      expect_domain, expect_mode, require_tools, ...
```

`Normalize()` 必须做的事：

1. 没有 `dialogue` 时，用 `setup_messages + message` 合成；最后一轮默认 `judge: true`。
2. 有 `dialogue` 但没有任何 `judge: true` 时，把最后一轮标为 judge。
3. 没有 `expect_intent` 时从扁平 `expect_domain/mode/sop` 填充。
4. 没有 `expect_execution` 时：有 `execution_profile` 用 profile，否则用 `require_tools`。
5. 有 rubric 且未配 judge 时，默认 `enabled=true, model_slot=auxiliary, min_score=0.7`。

`SyncLegacyUtterances()`：把 `dialogue` 再写回 `message` / `setup_messages`，给仍读旧字段的 runner。

**新代码只写 `dialogue` + 结构化 expect_*。扁平字段只为兼容。**

### 5.2 对话脚本

```json
[
  {"role": "user", "text": "先问一件事"},
  {"role": "user", "text": "再在同一 session 续问", "judge": true}
]
```

| 字段 | 含义 |
|------|------|
| `role` | 目前评测只驱动 `user` 轮；assistant 由真实 Loop 产生 |
| `text` | **完整自然句**。禁止半句话、禁止「可以」「这边呢」当独立轮 |
| `judge` | 本轮结束后做 Verify（通常最后一轮） |
| `on_clarify` | 不是普通续问，而是澄清回调/澄清跟进轮 |

形态只有三类，先选形态再写话术：

| 形态 | 脚本 | 测什么 |
|------|------|--------|
| **单轮 execute** | 一句完整请求 `[+ on_clarify 默认补全]` | 路由 + 必调工具 |
| **多轮上下文** | setup 轮（不 judge）+ 最后一轮 judge | 指代、切换对象、跨 domain 衔接 |
| **灰区澄清** | 首轮应 `mode=clarify`；用户选择写在 `on_clarify` | 该问清却直接执行 = fail |

### 5.3 澄清（Clarify）两种用法，不要混

评测里 Agent 缺槽位时会调用 clarify 工具。脚本侧有两种接法：

**A. 同轮 ClarifyFn（单轮 execute 缺参）**

- 普通用户轮发出后，Loop 内部触发 clarify。
- Runner 用 `PickClarifyAnswer(question, choices, defaults)` 自动选一项，**不增加用户轮次**。
- 仅当评判轮结束后 **必调工具仍缺失** 时，才把 `on_clarify` 文本再作为新用户轮发出（跟进轮）。

**B. 拆分脚本（灰区 / `mode=clarify`）**

- `expect_mode = clarify` 且 dialogue 里有 `on_clarify: true`。
- **intent 核的是第一轮** 路由（必须是澄清），不是选择之后的路由。
- `on_clarify` 轮始终发送，用来走完「用户做了选择」之后的回复质量。
- 同轮 ClarifyFn 关闭，避免和显式第二轮抢答。

`PickClarifyAnswer` 匹配顺序：

1. 默认文本与某个 choice 互相包含 → 用该 choice。
2. 无 choices → 用第一条默认文本。
3. 都匹配不上 → 视为无法自动澄清（不要瞎选）。

领域关键词匹配表由目标 Agent 自己维护；机制层只需要「默认文本 ↔ 选项」的包含/关键词对齐。

### 5.4 分类怎么切（与业务无关的切法）

按 **能力面 × 对话风险** 分组，不要按产品模块名堆：

| 建议分类 | 覆盖的机制 |
|----------|------------|
| 查询 / 分析 | 单轮 gather；同能力内的细 act |
| 执行类任务 | execute + 必调工具；缺参澄清 |
| 灰区 / 澄清 | 歧义、复合意图、必须先问清 |
| 闲聊 / QA | 不调业务工具；解释、事后评价 |
| 管理 / 列表 | 列出资源、查状态 |
| 历史 / 报告 | 依赖上一轮结果的 gather |
| 知识 / 检索 | 指定知识源 |

每类至少覆盖：

- 1 条 **显式单轮**（话术把槽位说全）
- 1 条 **口语/缺槽**（应澄清或应默认补全）
- 1 条 **多轮衔接**（先 A 再 B，或指代，或切换对象）

### 5.5 期望怎么写

**Intent（路由）**

```text
expect_intent: { domain, mode, sop, act? }
mode ∈ gather | execute | clarify | talk
sop：当前参考实现恒为 false（不走确定性 SOP 短路）
act：同一 domain 内的细分动作；没有细分就留空
```

灰区用例的 intent 期望是 **澄清本身**（`ambiguous/clarify`），不是用户选择之后的业务 domain。

**Execution（工具）**

优先引用 **ExecutionProfile ID**，不要每条 case 手写工具清单。

```text
ExecutionProfile
  id
  required[]     { tool, scope }
  forbid_on_judged_turn[]
  可选领域约束（例如禁止某条捷径组合）
```

`scope`：

| scope | 通过条件 |
|-------|----------|
| `judged_turn` | 评判轮必须出现该工具 |
| `session` | 整段 session 某处出现即可（允许 setup 轮已调） |
| `judged_turn_or_session` | 并集 |

没有 profile 时退回 legacy：`require_tools` 全在 judged turn，`forbid_tools` 评判轮不得出现。

**Reply（文本）**

| 字段 | 机制 |
|------|------|
| `min_reply_chars` | 默认 ≥ 20，挡空回复 |
| `must_cover[]` | 全部子串都要出现（大小写不敏感） |
| `must_not[]` | 任一出现即 fail |
| `pass_keywords[]` | 命中任一即可（旧字段，能不用就不用） |
| `rubric` | 给 LLM judge 的自然语言验收标准 |

rubric 写法：

- 写 **用户意图是否达成**，不写标准答案全文。
- 写清「不应该做什么」（误触发执行、该澄清却直接干）。
- 允许措辞不同。

---

## 6. Verify 管道（功能核心）

`VerifyLiveFull(session, options, judge)` 按固定顺序追加 check，**全部通过才算通过**：

```text
1. intent        路由快照 vs expect_intent
2. execution     profile / require / forbid（配了才跑）
3. reply_length  最短字符
4. must_cover    必现词
5. must_not      禁现词
6. llm_judge     仅当 rubric + judge.enabled
```

每条 check 的形状统一：

```text
{ type, passed, score?, detail, expected{}, actual{}, model? }
```

汇总：

- `passed` = 所有 check 的 AND
- `detail` = `intent=ok; execution=fail(...)` 这种拼接
- `summary` = `{intent_pass, execution_pass, reply_pass, judge_pass, judge_score, ...}`
- 落库：一次 run + 多条 `agent_eval_run_checks`

**intent 选哪一轮快照**

- 普通 / 多轮：`last_turn_plan`（最后一轮）。
- 拆分澄清脚本：`turn_plan_trace[0]`（第一轮必须是 clarify）。

**LLM judge**

- 独立辅助模型，低温，只输出 JSON：`{pass, score, reason, gaps}`。
- 输入：对话摘要、最后用户句、路由 detail、rubric、实际回复。
- `pass` 且 `score >= min_score`（默认 0.7）才算过。
- judge 失败（空回复、provider 不可用、解析失败）= 该 check fail，不要静默跳过。
- **judge 不能替代 intent/execution。** 路由错了、工具错了，即使文案漂亮也是 fail。

---

## 7. Live 执行器逻辑（Job / 单条）

一条 Live 用例的运行时序：

```text
load options → Normalize → SyncLegacy
DialogueExecutionPlan → regular[] + clarify[]
session_id = ""

for turn in regular:
    Agent.Run(session, turn.text, ClarifyFn? )
    session 复用同一 id

if NeedsClarifyFollowup(session, options):
    for turn in clarify:
        Agent.Run(session, turn.text, ClarifyFn 关闭)

VerifyLiveFull(session)
persist run + checks
```

Runner 侧环境（与人工聊天对齐，但去掉交互阻塞）：

- `metadata.source = "eval"`
- 自动 approval = true，plan gate = false
- 非 interactive；MCP/用户 token 从 Job 创建请求注入
- 单轮超时建议 10 分钟
- **全 Job 串行**（一把进程锁）。Live 共用同一个 Agent.Run，交错会串 session。

`NeedsClarifyFollowup`：

- 拆分澄清脚本 → 恒 true。
- 否则：配置了 `on_clarify` **且** legacy 必调工具在 session 里还缺 → true。

单条 HTTP verify 不负责发对话：客户端（Dock Chat 或 Job）必须已经跑完，只带 `session_id` 来校验。

---

## 8. 存储与 API

### 8.1 表

| 表 | 一行代表 |
|----|----------|
| `agent_eval_cases` | 用例定义 |
| `agent_eval_runs` | 一次 verify 结果（含 dialogue_snapshot / summary） |
| `agent_eval_run_checks` | 该次的每条 check |
| `agent_eval_suites` | 保存的「选了哪些分类/用例」 |
| `agent_eval_jobs` | 一次批量 Live |
| `agent_eval_job_items` | Job 里的一条用例进度 |

Job item 状态：`pending → running → pass | fail | error | cancelled`。  
Job 状态：`queued → running → pass | fail | cancelled | error`。有任意 item 非 pass 则 Job 为 fail。

### 8.2 HTTP 最小面

**用例 CRUD**

```text
GET    /v1/dashboard/eval/cases
GET    /v1/dashboard/eval/cases/{id}
POST   /v1/dashboard/eval/cases
PUT    /v1/dashboard/eval/cases/{id}
DELETE /v1/dashboard/eval/cases/{id}
```

列表返回时做 **enrich**：把 `options_json` 展开成 `dialogue` / `expect_reply` / `run_mode`，方便 UI。不要让前端自己猜遗留字段。

**Plan-only / Live verify**

```text
POST /v1/dashboard/eval/run-turn-plan          # 整份 plan-only 套件
POST /v1/dashboard/eval/cases/{id}/run         # 仅 plan_only；Live 返回 400
POST /v1/dashboard/eval/cases/{id}/verify      # body: {session_id}
```

**自动测评**

```text
GET  /v1/dashboard/eval/catalog
GET/POST /v1/dashboard/eval/suites
GET/POST /v1/dashboard/eval/jobs
GET  /v1/dashboard/eval/jobs/{id}
POST /v1/dashboard/eval/jobs/{id}/cancel
GET  /v1/dashboard/eval/jobs/{id}/items/{item_id}   # session 对话 + loop 步骤/报错
GET  /v1/dashboard/eval/auto                        # 可选 HTML 页
```

创建 Job：`{title, suite_id?, category_ids?, case_ids?, mcp_token?}`。  
展开规则：有 `case_ids` 用指定用例；否则按 `category_ids`；都空 = 全量。

运营 UI 只消费这套 API。不要在客户端再实现一套 Verify。

---

## 9. 代码生成与发布闭环

```text
internal/eval/cases.go          # 人改这里
        │
        ├─ gen_*_eval_sql.go        → schema.sql / postgres_eval.sql 的 seed
        └─ gen_*_cases_json.go      → scripts/eval/*.json 清单
                │
                ▼
        migrate_*_eval_db.py        → 同步到运行时 Postgres
                │
                ▼
        POST /jobs  或  远程冒烟脚本
```

规则：

- JSON / SQL seed **禁止手改**。生成器覆盖。
- 改用例后：单测 → 生成 → commit 生成物 → 部署 runtime → 迁移 DB → 再跑 Live。
- Plan-only 必须在 Live 之前绿：分类都错，跑 Live 是浪费。

---

## 10. 给其他 Agent 的落地步骤

每步一次会话。本步测试未绿，禁止下一步。

### Step E0 — 观测点

在 Session 上实现 `SyncLastTurnPlan` / `AppendTurnToolsTrace` 及读取函数。  
Loop 每轮结束必须调用。  
验收：单测里造一个 session，能读出 last plan 与 judged/session tools。

### Step E1 — 类型与 Normalize

实现 Dialogue / ExpectIntent / ExpectExecution / ExpectReply / Judge / CaseOptions。  
单测：遗留 `setup+message` ↔ `dialogue` 互转；缺 judge 标记时自动打在最后一轮。

### Step E2 — Plan-only

`RunSuite(planner)`：对每条 turn 调 `Plan(user_text, last_domain)`，比 domain/mode/act/sop。  
HTTP：`POST /eval/run-turn-plan`。  
验收：mock planner 能红能绿；真实 planner 接上后套件可跑。

### Step E3 — ExecutionProfile

用 **ID** 声明工具契约，Verify 只认 ID。  
先做 2 个 profile：一个「评判轮必调工具 X」，一个「session 内曾调过 Y 即可」。  
禁止在 case 里复制长工具列表（除非过渡期 legacy）。

### Step E4 — Verify 管道

实现 `VerifyLiveFull`。check 顺序固定。  
单测用 **内存 session + 假 trace**，不启动 LLM、不跑 Chat。  
覆盖：intent 错、缺工具、禁工具、短回复、must_cover 失败。

### Step E5 — LLM Judge

`ReplyJudge` 接口 + 一个 LLM 实现。不可用时 check fail。  
单测用 fake judge，不要打真实模型。

### Step E6 — Clarify 脚本

`DialogueExecutionPlan` / `NeedsClarifyFollowup` / `PickClarifyAnswer`。  
单测三种：无需跟进、缺工具才跟进、clarify-mode 强制跟进。

### Step E7 — Live Job

`runEvalJob` 串行；每 item 独立 session；自动 approval。  
`cases/{id}/run` 对 Live 返回 400。  
验收：用 mock Agent.Run 跑 2 条（1 pass 1 fail），Job 汇总正确。

### Step E8 — Catalog / Suite / 落库

DDL + cases CRUD + catalog 按分类计数。  
生成器：源码 → SQL seed + JSON。  
验收：改一条 case 的 title，重新生成后 JSON/SQL 都变。

### Step E9 — 运营面

一个「选分类 → 创建 Job → 轮询进度 → 点开 item 看对话与 checks」的页面即可。  
不要在这一步做双模型对比、随机实体、脚本可视化编辑（那些是增强，不是机制）。

### Step E10 — 填领域用例

用第 5.4 节的分类切法，为 **你的** domain 填表。  
每条必须同时有：自然话语、intent 期望、execution 期望、rubric。  
话术用完整句子。复合意图先走澄清类，不要指望模型一次做两件互斥的事。

---

## 11. 设计原则（实现时不得违反）

1. **结构化检查在前，语义裁判在后。** judge 是补漏，不是主裁判。
2. **路由来自 Planner 快照，不来自回复文本。**
3. **工具契约用 profile ID，** 避免 20 条 case 各写一份 require_tools。
4. **一条用例一个新 session。** 多轮只存在于该用例内部。
5. **Live 与 Plan-only 入口分离。** Live 必须先 Chat 再 verify。
6. **代码是源，DB 是副本。**
7. **Live Job 串行。**
8. **口语完整。** 评测话术 = 真实用户会说的整句。
9. **灰区必须可澄清。** 模糊输入的正确行为常常是 clarify，不是猜。
10. **失败要可诊断。** 每条 check 带 expected/actual；Job item 能回溯 session 与 loop 步骤。

---

## 12. 常见错误（看到就停）

| 错误 | 正确做法 |
|------|----------|
| 用关键词规则当生产 Planner，再用同一套规则评测 | 生产与评测都走 LLM Planner；评测断言的是输出结构 |
| Live 用例直接 `POST .../run` | 先对话，再 verify |
| 多条用例共用 session | `session_cleanup: before_run`，每 case 新 session |
| 只写 rubric，不写 intent/tools | 三层期望都要有 |
| 把 setup 轮的工具当成评判轮工具 | 用 profile 的 `session` / `judged_turn` 区分 |
| 澄清用例去核最后一轮 domain | 核第一轮是否 clarify |
| 并行跑两个 Live Job | 进程锁串行 |
| 在 SQL 里改话术 | 改源码再生成 |
| 半句续问当独立用户轮 | 写成完整自然句，或放进 setup 后的整句 |
| judge provider 挂了当 skip | 当 fail |

---

## 13. 参考实现对照（只指机制文件，不含领域话术）

落地时对照 GeeGooAgent 这些文件看 **怎么拆**，不要拷贝其中的领域句子：

| 机制 | 参考 |
|------|------|
| 类型 / Normalize | `internal/eval/evalcase_types.go` |
| 脚本与澄清 | `internal/eval/clarify_followup.go` |
| Live Verify | `internal/eval/verify_live.go` |
| 工具契约校验 | `internal/eval/execution_verify.go`、`internal/domaincatalog/execution_profile.go` |
| Plan-only | `internal/eval/turnplan.go`、`turnplan_report.go` |
| Judge | `internal/eval/reply_judge.go` |
| Session 观测点 | `internal/chatsession/turnplan_meta.go` |
| HTTP / Job | `internal/runtimeapi/dashboard_eval*.go` |
| DDL | `internal/infra/pgschema/postgres_eval.sql` |
| 生成器 | `scripts/gen_turnplan_eval_sql.go`、`scripts/eval/gen_turnplan_cases_json.go` |
| 运营 UI | `trading_operation` 自动测评 Tab；runtime `GET /v1/dashboard/eval/auto` |

领域套件（话术、分类名、业务工具）各自维护，**不要写进本指南，也不要写进新 Agent 的机制层。**

---

## 14. 完成清单

机制完成的标准（与业务无关）：

- [ ] Loop 每轮写入 `last_turn_plan` 与 `turn_tools_trace`
- [ ] Plan-only 套件 HTTP 可跑，结果按 turn 列出
- [ ] Live verify 拒绝无 `session_id`；Live `run` 返回 400
- [ ] Verify 至少产出 intent / execution / reply / judge 四类 check（后两类按配置可缺）
- [ ] ExecutionProfile 用 ID 引用
- [ ] 澄清：同轮 ClarifyFn 与拆分脚本行为有单测
- [ ] Job 串行、可取消、item 能看到 session 对话
- [ ] 用例源码 → SQL/JSON 生成器 → DB 迁移闭环存在
- [ ] 运营页能选分类并创建 Job
- [ ] 领域话术不进 `eval` 机制文件（只进 cases 数据文件）
