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
| `dev_ensure_archive` | 确保策略档案存在：有则读取注入，无则自动生成 |
| `summarize` | 输出已载入/生成的档案摘要 |

## 触发示例（推荐首句）

- 「**读取 Macd4H 策略**」
- 「读取 SAR+MACD 策略」

兼容旧说法：「策略开发 Macd4H」

## 与「生成策略档案」的关系

- 单独说「帮我生成 Macd4H 的策略档案」→ `generate_strategy_archive` workflow
- 「读取 Macd4H 策略」→ 本 workflow；**不要求**用户事先手动生成档案
