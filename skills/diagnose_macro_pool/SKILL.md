---
name: diagnose_macro_pool
description: 宏观监控池 JEV 合成（HTTP skill，workflow 负责 PUT）
version: "1.0.0"
---

# diagnose_macro_pool

workflow 在当日同池、同 `source_batch_id` 下齐套 success 标的后组包调用。skill 仅：

1. 校验 `symbols[]`（`result_id`、方向、理由）
2. GeeGooData **Choice** 池级方向（禁止 LLM 计票）
3. Decision **chat** 生成简体中文池级 `jev_reason`

**不会** GET canonical 表、**不会** PUT `/macro/jev/results/pools/*`。

## 触发

`POST /v1/skills/run`

```json
{
  "skill": "diagnose_macro_pool",
  "macro_pool": {
    "date": "2026-11-20",
    "pool_workflow": "crypto",
    "source_batch_id": "macro-batch-run-id",
    "title": "加密",
    "symbols": [
      {
        "symbol": "BTC-USD.CRYPTO",
        "result_id": "sym:…",
        "jev_direction": "long",
        "jev_reason": "…",
        "confidence": 0.82
      }
    ]
  }
}
```

`result.jev_direction` 为 ledger：`long` / `short` / `neutral`；`detail_ref=diagnose_macro_pool`。