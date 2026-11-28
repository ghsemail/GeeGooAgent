---
name: diagnose_macro_news
description: 宏观监控池新闻分析与 JEV（HTTP skill，news-worker 负责落库）
version: "2.0.0"
---

# diagnose_macro_news

GeeGooData `news-worker` 只把评为 `validity=news` 且 `materiality=high` 的新闻写入本节点 `aidb.macro_news`：先调用本 skill 判断价值（条目层），再用已保存的当日高价值新闻调用本 skill 做单池结论（`rollup_only`）。跨池结论是独立的 `diagnose_macro_news_overview`。

三层共用同一组 JEV 原则（`strategydev/lab_macro_news_prompts.go`）：池级与跨池方向只来自 JEV Choice；综合模型只写理由，不改方向；反向信息由模型权衡；只有没有可判断的信息才是不明确；调用失败是错误，不是中性。条目层的方向由条目分析模型给出。

条目层：

1. 校验 `pool_id`（七个监控池之一）与 `items[]`（最多 40 条，`news_id` 唯一，标题非空）
2. LLM 逐条先判断是否新闻、再判断对本池的重要性。**只有 `news` + `high`** 才输出中英标题与摘要、`attitude`、`jev_direction`、`jev_reason`、`theme`、`tags`、`related_symbols`；其他条目只返回价值字段（主模型失败时只允许一个备选模型）
3. URL 规则命中的行情页由调用方直接丢弃，不会发给本 skill

单池层：

1. GeeGooData Decision **Choice** 决定池级方向（禁止 LLM 或代码计票）
2. 综合模型（主模型加一个备选）写全池结论 `jev_reason`：简体中文 Markdown，以「## 整体结论」开头。输入为最多 16 条支持方向的新闻与 8 条其他新闻
3. 若两个模型都没有给出可用结论：保留 Choice 方向，`jev_reason` 写为「理由总结失败：…」，`reason_status=failed`（调用方下次重试）。不再使用固定句式的标题列表代替总结

**不写** Mongo，**不调用** GeeGooSignal（含 analyze-api 与 `/macro/news*`），**不取行情价格**（只看新闻）。落库由调用方完成。

## 时间预算

调用方与 agent BFF 超时 15 分钟。单次模型调用（主 + 备）最长 210s 且不可被 context 取消，Decision 调用最长 90s。

| 调用 | 上限 |
|---|---|
| 条目（`skip_pool_jev`） | 600s 后不再发起新的模型调用，最长 810s；超时条目返回 `error_code=budget_exhausted`，调用方留到下次窗口 |
| 单池（`rollup_only`） | 90s + 210s = 300s |
| 跨池 overview | 90s + 210s = 300s |
| 条目 + 单池同一次调用 | 条目 270s 后停止，合计最长 780s |

## 触发

`POST /v1/skills/run`

```json
{
  "skill": "diagnose_macro_news",
  "macro_news": {
    "date": "2026-11-27",
    "pool_id": "cn_a",
    "pool_label": "A股池",
    "run_id": "macro-news-refresh-…",
    "items": [
      {
        "news_id": "…",
        "title": "…",
        "snippet": "…",
        "source_id": "sina_roll",
        "source_label": "新浪财经",
        "url": "https://…",
        "published_at": "2026-11-26T15:04:05+08:00",
        "matched_keywords": ["上证指数", "北向资金"]
      }
    ],
    "prior_verdicts": [
      {"news_id": "…", "title": "…", "attitude": "Bearish", "jev_direction": "short", "jev_reason": "…"}
    ],
    "skip_pool_jev": false
  }
}
```

单次最多 40 条。调用方的用法：

- 条目批次：带 `items`，设 `skip_pool_jev: true`。
- 单池结论：`rollup_only: true`，`items` 为空，`prior_verdicts` 为该池当日全部已保存的高价值条目（1–2000 条）。
- 不带两个标志时，同一次调用内分析条目并对其中的高价值条目与 `prior_verdicts` 做单池结论。

## 返回

`{"status":"ok","skill":"diagnose_macro_news","result":{…}}`

| 字段 | 说明 |
|---|---|
| `result.items[]` | 与请求同序。`analysis_status` ∈ `ok`/`failed`；失败时 `error_code` 可能为 `budget_exhausted` 或 `unrated` |
| `attitude` / `jev_direction` | 仅高价值条目：`Bullish`/`Bearish`/`Neutral`，`long`/`short`/`neutral`；其他条目为空（不是 Neutral） |
| `summary_cn` / `summary_en` | 仅高价值条目；分节并带情绪行 |
| `theme` | `rates_central_bank`/`growth_inflation`/`geopolitics_risk`/`sector_theme`/`equity_etf_catalyst`/`other` |
| `related_symbols[]` | `{symbol,name,market}`。模型须引用原文片段 `mention`，代码丢弃片段不在标题或摘要里的标的；调用方归一代码并补全名称 |
| `result.pool_status` | `ok`；`pending`（没有高价值条目、成功条目不足或 Choice 失败，见 `pool_error`）；`skipped` |
| `result.jev_direction` / `jev_reason` / `reason_status` / `confidence` / `choices` / `conflict` | 池级结论；非 `ok` 时方向与理由为空 |

## 新闻价值（v1）

与方向（neutral 只表示方向不明）和 `analysis_status`（只表示处理是否完成）相互独立。

| 字段 | 取值 |
|---|---|
| `validity` | `news` / `non_news` |
| `non_news_type` | `quote_page` / `listing` / `data_table` / `promo` / `other`，仅 `non_news` |
| `materiality` | `high` / `medium` / `low` / `none`（真实新闻但与本池无关） |
| `value_reason` | 一句简体中文 |
| `value_source` / `value_version` | `model` / `v1`；未评级为空 |
| `pool_eligible` | 由代码计算，忽略模型给出的值 |

- 每条都必须有完整评级。缺失或非法视为该次回复不可用（会尝试备选模型）；仍未评级则该条 `failed`、`error_code=unrated`。
- 模型对高价值条目只给出 `attitude` 或 `jev_direction` 时按 Bullish↔long、Bearish↔short、Neutral↔neutral 补齐；给了非法值或两者都缺，该条记 `failed`，不编造方向。
- 单池结论只使用高价值条目与 `prior_verdicts`。门槛：成功条目数 ≥ `min(3, ⌈总数/2⌉)`，总数含 `prior_verdicts`、不含被判为非高价值的条目。
- 整个请求失败（如 LLM 未配置）返回非 200，调用方在下次运行重试。

## 部署

本版本的返回格式与 `rollup_only` 需要同版本的 GeeGooData：两者须一起部署，避开 news-worker 的运行时刻。
