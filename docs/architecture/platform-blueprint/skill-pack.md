# Skill Pack 规范

> L5 扩展点。新 Agent 的**业务差异**应只体现在 `skills/` + Tool Catalog，不改 L4 核心。

---

## 目录结构（必须）

```text
skills/<skill_name>/
├── SKILL.md                 # YAML frontmatter + 人类可读说明
├── manifest.yaml            # 机器可读 SSOT（Phase 3 loader 读此文件）
├── workflow.md              # 业务步骤（禁止写 HTTP URL，引用 Tool 名）
├── template.md              # 输出模板（报告/邮件/JSON）
└── supervisor_checks.yaml   # 跑完后的验收断言
```

---

## SKILL.md frontmatter

```yaml
---
name: first_skill
description: One-line description for agent discovery.
version: "1.0.0"
---
```

正文须包含：触发方式、`geegoo run` 等价命令、非满足条件时的 short-circuit 行为。

---

## manifest.yaml SSOT

完整 schema（智能体生成首个 Skill 时**原样创建再改字段**）：

```yaml
name: first_skill
version: "1.0.0"
description: 首个批量 Skill — Phase 1 MVP
phase: 1
mode: scheduled          # scheduled | interactive | signal

tools:                   # Tool 白名单（Registry 中必须已注册）
  - check_gate
  - list_work_items
  - fetch_context
  - synthesize_output
  - persist_result
  - write_execution_log

llm_tasks:               # 非 Registry 条目，窄 LLM 任务（可选）
  - synthesize_output

rules:                   # 相对仓库根路径
  - rules/api-routing.md
  - rules/output-format.md

workflow:
  prelude:               # Phase A 前置（全局）
    - id: check_gate
      tool: check_gate
      short_circuit_if: "gate_ok == false"
    - id: list_work_items
      tool: list_work_items

  phase_a:               # Phase A 主体
    - id: fetch_global
      tool: fetch_context
      params:
        scope: global

  phase_b:               # Phase B — 对每个 work item 重复
    per_item:
      - id: fetch_item
        tool: fetch_context
      - id: synthesize
        tool: synthesize_output
      - id: persist
        tool: persist_result

  meta:
    - id: execution_log
      tool: write_execution_log
      after_each_step: true

entities:                # Phase B 迭代对象（可选，loader 注入）
  source: list_work_items  # 从哪个 Tool 的 working 字段取列表
  id_field: code
```

### 字段说明

| 字段 | 必填 | 说明 |
|------|------|------|
| `mode` | 是 | 决定 Tool 过滤策略 |
| `tools` | 是 | 本 Skill 可用 Tool 子集 |
| `workflow.prelude` | 推荐 | 全局 gate + 列表 |
| `workflow.phase_a` | 视业务 | 不逐 item 的步骤 |
| `workflow.phase_b.per_item` | 视业务 | 逐 item 步骤 |
| `short_circuit_if` | 可选 | 表达式，false 时整 workflow 完成 |
| `llm_tasks` | 可选 | 文档对齐；实现可在 Bespoke Tool 内调 Gateway |

---

## supervisor_checks.yaml

```yaml
checks:
  - id: gate_respected
    description: 非满足条件时不应产生 item 级副作用
    when: "working.gate_ok == false"
    assert: "run.status == completed"
    assert_not: "any(working.items, status == persisted)"

  - id: all_items_done
    description: Phase B 结束后每个 item 应有终态
    when: "working.phase == done"
    assert: "all(working.items, status in [done, skipped])"
```

Phase 1：文件存在 + 人工对照。  
Phase 3：`Supervisor.verify()` 自动执行。

---

## template.md

输出模板的占位符**必须与 Working 字段名一致**：

```markdown
# 报告 — {{stock_name}} ({{code}})

## 摘要
{{summary}}

## 详情
{{details}}
```

Workflow 中 `synthesize` 步骤负责填充 Working 或直出 content。

---

## bundled 子 Skill（可选）

```yaml
bundled:
  - skills/bundled/news-fetcher
```

子目录含独立 `SKILL.md`，供 Bespoke Tool 内部调用，不暴露给主 Skill 的 LLM tool list。

---

## 新 Skill 自检表

- [ ] `manifest.yaml` 中每个 `tool` 已在 Registry 注册
- [ ] `mode: scheduled` 未包含 mutating Tool
- [ ] `workflow.md` 无 hardcode URL
- [ ] 每个 Tool 有对应 `Working.Apply` 分支
- [ ] `supervisor_checks.yaml` 至少 2 条
- [ ] dry-run 跑通：`run <skill> --dry-run`
- [ ] 中断后续跑：`resume --session <id>`

---

## GeeGoo 参考

见 [skills/pre_market/manifest.yaml](../../../skills/pre_market/manifest.yaml)（完整盘前示例）。
