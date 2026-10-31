---
name: generate_strategy_archive
description: 生成策略档案：读策略库 → LLM 按 template 合成 Agent 档案 → save_strategy_knowledge → search_knowledge 读回。Chat 用 skill_view 加载后按步调普通工具。
kind: workflow
status: available
trigger_modes: chat
---

# 生成策略档案（generate_strategy_archive）

从 GeeGoo 策略库读取定义，由 LLM 结合模型常识合成 **策略档案** 文档，写入 WeKnora `策略档案/` 目录。

## 何时使用

- 「帮我生成 Macd4H 的策略档案」
- 「生成策略档案 SAR+MACD」
- 「了解一下共振策略，整理到知识库」

兼容旧说法「生成策略认知 / 策略认知」仍可触发；读回时会同时检索新旧目录名。

## Chat 执行顺序（不要用 run_chat_workflow_turn）

**推荐首步（非硬性）**：从注入的 **Skills catalog** 选中 `generate_strategy_archive` 后，先 **skill_view**(`skill=generate_strategy_archive`) 加载本 SKILL.md；进入合成前再 **skill_view**(`skill=generate_strategy_archive`, `file=template.md`) 读取落库骨架（Eval Live 不强制本步）。

1. **解析意图**（`cognition_pick`）：从用户话里提取 `strategy_query`（策略/catalog 名）。缺名称 → **clarify**。
2. **模糊匹配（按需）**：用户只给指标/模糊信号名、未指定具体 catalog 条目 → **search_signals**(`q=关键词`) 列候选；2+ 命中 → **clarify**(`choices=各 name`，最多 4 个）；禁止未搜索就猜 `signal_id`。
3. **读策略库**（`cognition_read_catalog`）：按名称在策略库解析一条 catalog 行（顺序与 Workflow Runner 一致）：
   - **get_signal_combinations** — 组合策略
   - **get_index_signals** — 单指标信号
   - **get_custom_signal_for_skill** — 定制信号（需 `mcp_token`）
   - **get_custom_strategy_definitions** — 定制 definition  
   从命中行保留完整 JSON（含 `buy_signal`、`sell_signal`、`frequency`、`signal_id` 等），作为合成权威来源。
4. **合成档案**（`cognition_compose`）：依同目录 [`template.md`](template.md) 各节骨架 + 上一步策略库 JSON + 模型常识，写 Markdown 正文；frontmatter 含 `doc_type: strategy_agent_archive`；各节「>」写作要求只用于生成，**剥离后**再落库。
5. **写入知识库**（`cognition_save_kb`）：**save_strategy_knowledge**（`strategy_name`=展示名，`content`=正文，`folder_path=策略档案`）；记录返回的 `knowledge_id` / `title`。
6. **读回验证**（`cognition_verify_kb`）：**search_knowledge**（`query={策略名} 策略档案`，可选 `folder_path=策略档案`；兼容时会搜 `策略认知/`）；命中正文应含 `doc_type: strategy_agent_archive` 或「一句话定位」等档案特征。
7. **summarize**：向用户输出 Markdown 完成报告（策略库类型、知识库链接、读回验证片段）。

## 工具参数要点

| 步骤 | 工具 |
| --- | --- |
| 加载步骤 | skill_view |
| 模糊搜策略 | search_signals |
| 缺参 / 多选 | clarify |
| 读组合 | get_signal_combinations |
| 读单指标 | get_index_signals |
| 读定制 | get_custom_signal_for_skill |
| 读 definition | get_custom_strategy_definitions |
| 持久化 | save_strategy_knowledge |
| 读回 | search_knowledge |

## Phases（与上表步骤对应）

| Phase | 对应 Chat 步骤 |
| --- | --- |
| `cognition_pick` | 1 |
| `cognition_read_catalog` | 2–3 |
| `cognition_compose` | 4 |
| `cognition_save_kb` | 5 |
| `cognition_verify_kb` | 6–7 |

## 输出模板

最终写入知识库的正文骨架是同目录 [`template.md`](template.md)。各节「>」是写作要求：合成时交给 LLM，落库时剥离，不会写进档案正文。

## 知识库目录

| 目录 | 用途 |
|------|------|
| `策略资料/` | 人工上传的参考 PDF、外部策略文档（原 `策略/`） |
| `策略档案/` | 本 workflow 写入的 Agent 策略档案（原 `策略认知/`） |

## 与策略开发的关系

- 单独「生成策略档案」→ 本 skill（上表步骤）。
- 「读取 Macd4H 策略」→ **strategy_dev** workflow；无档案时会自动生成，不要求用户事先手动跑本 skill。
