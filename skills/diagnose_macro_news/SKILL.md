---
name: diagnose_macro_news
description: 宏观监控池新闻分析与 JEV（HTTP skill，news-worker 负责落库）
version: "1.0.0"
---

# diagnose_macro_news

GeeGooData `news-worker` 先把关键字命中的新闻写入本节点 `aidb.macro_news`，再调用本 skill。skill 仅：

1. 校验 `pool_id`（七个监控池之一）与 `items[]`（1–40 条，`news_id` 唯一，标题非空）
2. LLM 逐条输出中英标题与摘要、`attitude`、`jev_direction`、`jev_reason`、`theme`、`tags`（主模型失败时只允许一个备选模型）
3. GeeGooData Decision **Choice** 决定池级方向（禁止 LLM 或代码计票）
4. GeeGooData Decision **chat** 生成简体中文池级 `jev_reason`，引用最有力的新闻标题

**不写** Mongo，**不调用** GeeGooSignal（含 analyze-api 与 `/macro/news*`），**不取行情价格**（只看新闻）。落库由调用方完成。

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

单次最多 40 条。条目超过 40 时调用方分批：

- 非最后一批设 `skip_pool_jev: true`（可选，默认 `false`）。
- 最后一批在 `prior_verdicts`（可选，最多 2000 条）带上同一 `pool_id` + `date` 已分析完的条目，池级 JEV 按全集合成；这些条目不会重新分析。

## 返回

`{"status":"ok","skill":"diagnose_macro_news","result":{…}}`

| 字段 | 说明 |
|---|---|
| `result.items[]` | 与请求同序。`attitude` ∈ `Bullish`/`Bearish`/`Neutral`；`jev_direction` ∈ `long`/`short`/`neutral`；`analysis_status` ∈ `ok`/`failed` |
| `summary_cn` | 含「新闻内容」「新闻分析」两节及「市场情绪：看多\|看空\|中性」 |
| `summary_en` | 含 “News Summary”“News Analysis” 及 “Sentiment: Bullish\|Bearish\|Neutral” |
| `theme` | `rates_central_bank`/`growth_inflation`/`geopolitics_risk`/`sector_theme`/`equity_etf_catalyst`/`other` |
| `result.pool_status` | `ok`；`pending`（成功条目不足或 Decision 失败，见 `pool_error`）；`skipped` |
| `result.jev_direction` / `jev_reason` / `confidence` / `choices` / `conflict` | 池级结论；非 `ok` 时方向与理由为空 |

规则：

- 模型只给出 `attitude` 或只给出 `jev_direction` 时按 Bullish↔long、Bearish↔short、Neutral↔neutral 补齐；给了非法值或两者都缺，该条记 `failed`，不编造方向。
- 池级合成门槛：成功条目数 ≥ `min(3, ⌈总数/2⌉)`，总数含 `prior_verdicts`。
- 整个请求失败（如 LLM 未配置）返回非 200，调用方保留原始新闻并在下次重试。
