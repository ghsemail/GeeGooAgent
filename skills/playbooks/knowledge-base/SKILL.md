---
name: knowledge-base
description: 知识库、策略文档、查库、WeKnora、文档里怎么写、MACD 策略。用户说「按知识库」「查库」「策略文档怎么写」时触发。
---

# 知识库检索（按需）

本 skill **不是**行情 API。只有用户明确要按知识库 / 策略文档 / WeKnora 文档回答时才使用。

## 适用 Toolset

`knowledge`（默认 chat **不含**；本 skill 命中后本轮才会注入 `search_knowledge`）

## 标准流程

1. **先检索** — 调用 `search_knowledge`，`query` 用用户原问句的关键短语；若提到目录（如「策略」）再传 `folder_path`
2. **只根据命中片段回答** — 引用 `filename` / `folder`；不要扩写成未检索到的内容
3. **未命中** — 如实说知识库没有相关片段，并建议到 WeKnora 网页核对文档；不要改用 `get_current_price` / `get_mcp_analysis` 凑答案

## 硬规则

- 禁止把知识库当行情、持仓、报告接口
- 禁止在未调用 `search_knowledge` 的情况下声称「文档里写了…」
- 检索失败或零命中必须明确说明，不编造策略规则

## 反模式

- 用户只问股价 / 涨跌 / 分析个股 → **不要**用本 skill 的工具（走 `stock-analysis`）
- 把 WeKnora 片段和实时行情混成一条没有出处的结论
