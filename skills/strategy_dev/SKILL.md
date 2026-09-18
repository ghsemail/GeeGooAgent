---
name: strategy_dev
description: 策略开发主 Workflow。Step 1 策略认知：读策略库 → Web 搜索补充 → 写入知识库 → 读回验证。
kind: workflow
status: available
trigger_modes: chat
---

# 策略开发（strategy_dev）

Chat 触发的**主 Workflow**，按步骤串行执行策略开发任务。

## Step 1 · 策略认知

| Phase | 说明 |
|-------|------|
| `cognition_pick` | 解析用户指定的策略名称 |
| `cognition_read_catalog` | 从策略库读取完整定义（组合 / 指标 / 定制 / definitions） |
| `cognition_web_research` | `web_search` 补充公开资料 |
| `cognition_compose` | 合成 Markdown 认知文档 |
| `cognition_save_kb` | `save_strategy_knowledge` 写入 WeKnora（`策略认知/`） |
| `cognition_verify_kb` | `search_knowledge` 读回验证 |

## 触发示例

- 「策略认知 Macd4H」
- 「策略开发：了解一下共振策略，整理到知识库」
- 「学习策略 SAR+MACD 组合」

## 续跑

发送「继续」或 `POST /v1/chat/workflow/resume`（`action=continue_all`）。
