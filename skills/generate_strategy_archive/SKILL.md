---
name: generate_strategy_archive
description: 生成策略档案 Workflow。读策略库 → LLM 合成 Agent 可注入档案 → 写入知识库 → 读回验证。
kind: workflow
status: available
trigger_modes: chat
---

# 生成策略档案（generate_strategy_archive）

从 GeeGoo 策略库读取定义，由 LLM 结合模型常识合成 **策略档案** 文档，写入 WeKnora `策略档案/` 目录。

## Phases

| Phase | 说明 |
|-------|------|
| `cognition_pick` | 解析策略名称 |
| `cognition_read_catalog` | 读策略库（组合 / 指标 / 定制 / definitions） |
| `cognition_compose` | LLM 合成 Markdown（适用场景、参数、信号逻辑、Agent 指引） |
| `cognition_save_kb` | `save_strategy_knowledge` 写入知识库 |
| `cognition_verify_kb` | `search_knowledge` 读回验证 |

## 触发示例

- 「帮我生成 Macd4H 的策略档案」
- 「生成策略档案 SAR+MACD」
- 「了解一下共振策略，整理到知识库」

## 知识库目录

| 目录 | 用途 |
|------|------|
| `策略资料/` | 人工上传的参考 PDF、外部策略文档（原 `策略/`） |
| `策略档案/` | 本 workflow 写入的 Agent 策略档案（原 `策略认知/`） |

## 兼容

旧说法「生成策略认知 / 策略认知」仍可触发；读回时会同时检索新旧目录名。
