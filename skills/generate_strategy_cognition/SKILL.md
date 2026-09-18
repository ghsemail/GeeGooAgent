---
name: generate_strategy_cognition
description: 生成策略认知 Workflow。读策略库 → LLM 合成 Agent 可注入认知 → 写入知识库 → 读回验证。
kind: workflow
status: available
trigger_modes: chat
---

# 生成策略认知（generate_strategy_cognition）

从 GeeGoo 策略库读取定义，由 LLM 结合模型常识合成 **Agent 策略认知** 文档，写入 WeKnora `策略认知/` 目录。

## Phases

| Phase | 说明 |
|-------|------|
| `cognition_pick` | 解析策略名称 |
| `cognition_read_catalog` | 读策略库（组合 / 指标 / 定制 / definitions） |
| `cognition_compose` | LLM 合成 Markdown（适用场景、参数、信号逻辑、Agent 指引） |
| `cognition_save_kb` | `save_strategy_knowledge` 写入知识库 |
| `cognition_verify_kb` | `search_knowledge` 读回验证 |

## 触发示例

- 「生成策略认知 Macd4H」
- 「策略认知 SAR+MACD」
- 「了解一下共振策略，整理到知识库」
