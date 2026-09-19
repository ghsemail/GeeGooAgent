---
name: strategy_dev
description: 策略开发 Workflow。从知识库读取已有策略档案，作为开发上下文（后续 Step 扩展回测/调参等）。
kind: workflow
status: available
trigger_modes: chat
---

# 策略开发（strategy_dev）

在已有 **策略档案** 知识库条目基础上进行策略开发。第一步从 WeKnora `策略档案/` 目录读取档案文档（兼容历史 `策略认知/`）。

## Phases

| Phase | 说明 |
|-------|------|
| `dev_pick` | 解析策略名称 |
| `dev_read_cognition` | `search_knowledge` 读取策略档案；未命中则提示先生成档案 |
| `summarize` | 输出已载入的档案摘要 |

## 触发示例

- 「策略开发 Macd4H」
- 「策略开发：基于 SAR+MACD 组合继续」

## 前置条件

知识库中需已有对应策略的档案文档。若无，请先使用 **生成策略档案** workflow，例如：「帮我生成 Macd4H 的策略档案」。
