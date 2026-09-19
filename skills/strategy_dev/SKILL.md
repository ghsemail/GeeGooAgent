---
name: strategy_dev
description: 策略开发 Workflow。确保策略档案存在（无则自动生成），注入开发上下文（后续 Step 扩展回测/调参等）。
kind: workflow
status: available
trigger_modes: chat
---

# 策略开发（strategy_dev）

基于 **策略档案** 进行策略开发。第一步自动确保知识库中有对应档案。

## Phases

| Phase | 说明 |
|-------|------|
| `dev_pick` | 解析策略名称 |
| `dev_ensure_archive` | 查 `策略档案/`（兼容 `策略认知/`）：**有则读取注入**；**无则 inline 执行生成策略档案**（读策略库 → LLM 合成 → 写入 → 读回） |
| `summarize` | 输出已载入/生成的档案摘要 |

## 触发示例

- 「策略开发 Macd4H」
- 「策略开发：基于 SAR+MACD 组合继续」

## 与「生成策略档案」的关系

- 单独说「帮我生成 Macd4H 的策略档案」→ `generate_strategy_cognition` workflow
- 「策略开发 Macd4H」→ 本 workflow；**不要求**用户事先手动生成档案
