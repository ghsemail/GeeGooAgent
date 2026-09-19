<!--
  本文件是 generate_strategy_archive 的最终输出骨架。
  {{占位符}} 运行时填充；各节「>」是写作要求，只给 LLM / 预览用，写入知识库时会去掉。
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

> 供 Agent 注入上下文。执行信号以策略库为准，本文只解释、不另立规则。

## 一句话定位

> 1–2 句：做什么、核心指标、默认周期。不要复述 JSON。

{{一句话定位}}

## 适用场景

> 适合的行情（趋势/震荡）、周期、标的特征，写 2–4 条。

{{适用场景}}

## 不适用 / 风险

> 失效市况、误报场景、滑点或参数敏感。不要编造策略库没有的限制。

{{不适用 / 风险}}

## 参数说明

> 只解释策略库已有 param：含义，以及调大/调小会怎样。

{{参数说明}}

## 指标与信号逻辑

> 说明 buy/sell 里各 index 如何组合：signal 触发还是 flag 过滤，AND 还是 OR。

{{指标与信号逻辑}}

## 买卖规则（与策略库对齐）

> 对齐 buy_signal / sell_signal。禁止新增入场或出场条件。

{{买卖规则（与策略库对齐）}}

## 周期与频率

> 写 frequency 和建议回看窗口；没有就写「以策略库为准」。

{{周期与频率}}

## Agent 使用指引

> 用户问「是什么 / 怎么用 / 什么参数」时引用本档案；probe 前确认 code、frequency、months_back。

{{Agent 使用指引}}

---

## 附录 · 策略库原文

```json
{{catalog_json}}
```
