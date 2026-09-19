<!--
  本文件是 generate_strategy_archive 的最终输出模板。
  约定：skills/<workflow>/template.md = 该 workflow 写入知识库 / 报告的正文骨架。
  占位符由运行时填充；带 ## 的章节（附录除外）是 LLM 必须逐字输出的标题。
-->

---
doc_type: strategy_agent_archive
strategy_name: {{strategy_name}}
catalog_type: {{catalog_type}}
signal_id: {{signal_id}}
generated_at: {{generated_at}}
source: geegoo_catalog + llm
agent_use: inject when user asks about this strategy, before probe/backtest
---

# {{strategy_name}} · 策略档案

> 供 Agent 注入上下文；**执行信号以策略库为准**。

## 一句话定位

{{section:一句话定位}}

## 适用场景

{{section:适用场景}}

## 不适用 / 风险

{{section:不适用 / 风险}}

## 参数说明

{{section:参数说明}}

## 指标与信号逻辑

{{section:指标与信号逻辑}}

## 买卖规则（与策略库对齐）

{{section:买卖规则（与策略库对齐）}}

## 周期与频率

{{section:周期与频率}}

## Agent 使用指引

{{section:Agent 使用指引}}

---

## 附录 · 策略库原文

```json
{{catalog_json}}
```
