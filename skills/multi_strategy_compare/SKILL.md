---
name: multi_strategy_compare
description: 固定标的，串行 probe 多个策略并输出买/卖次对比表。支持 Chat 关键词触发与 Scheduler cron。
---

# 多策略信号对比 Workflow

## 触发

- **Chat**：多策略、对比、挨个跑、依次测
- **Cron**：`jobs.json` 中配置 `skill=multi_strategy_compare` + `prompt`（含标的与策略名）

## 阶段

1. `resolve_symbol` — 解析标的
2. `pick_strategies` — 从用户句或 catalog 选取策略
3. `probe_foreach` — 串行 `probe_bot_signal_series`
4. `summarize` — 输出对比 Markdown 表

## 续跑

Chat 会话中发送「继续」「重试失败」，或 `POST /v1/chat/workflow/resume`。
