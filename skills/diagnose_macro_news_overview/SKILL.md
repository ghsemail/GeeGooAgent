---
name: diagnose_macro_news_overview
description: 当日跨池宏观新闻结论（HTTP skill，news-worker 负责落库）
version: "1.0.0"
---

# diagnose_macro_news_overview

宏观新闻三层中的第三层（条目 → 单池 → 跨池）。它与 `diagnose_macro`、`diagnose_macro_overall` 无关，不读行情价格，**不写** Mongo。GeeGooData `news-worker` 把结果按 `run_date + node_id` 存入 `aidb.macro_news_overview`。

1. 输入为当日各池已完成的单池结论与少量条目摘要，**不传**新闻全文
2. GeeGooData Decision **Choice** 决定整体方向（看多 = risk-on，看空 = risk-off；波动率池的看多表示波动率上行）
3. 综合模型（主模型加一个备选）写以「## 整体结论」开头的简体中文 Markdown；模型不能改方向
4. 总结失败时保留方向，`jev_reason` 为「理由总结失败：…」，`reason_status=failed`
5. Choice 失败时请求失败（非 200），不返回中性

时间预算：Choice 90s + 总结 210s = 最长 300s。

## 触发

`POST /v1/skills/run`

```json
{
  "skill": "diagnose_macro_news_overview",
  "macro_news_overview": {
    "date": "2026-11-27",
    "node_id": "us-hk",
    "run_id": "macro-news-refresh-…",
    "pools": [
      {
        "pool_id": "hk", "pool_label": "港股池",
        "jev_direction": "short", "jev_reason": "## 整体结论…",
        "bullish": 1, "bearish": 3, "neutral": 0, "high_value_count": 4,
        "items": [{"title": "…", "jev_direction": "short", "jev_reason": "…"}]
      }
    ],
    "missing_pools": ["cn_a", "crypto"]
  }
}
```

- `pools`：1–7 个，`pool_id` 唯一且 `jev_direction` ∈ `long`/`short`/`neutral`；每池最多 12 条摘要
- `missing_pools`：当日没有结论的池，不阻塞生成，结论中会注明

## 返回

`{"status":"ok","skill":"diagnose_macro_news_overview","result":{date,node_id,run_id,jev_direction,jev_reason,reason_status,confidence,choices,conflict,model,missing_pools}}`
