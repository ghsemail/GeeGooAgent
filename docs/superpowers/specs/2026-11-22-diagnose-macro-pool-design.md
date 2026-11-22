# diagnose_macro_pool：宏观监控池 JEV 合成 skill

**日期:** 2026-11-22  
**状态:** 实施中  
**范围仓库:** GeeGooAgent（skill + 日批 workflow）；GeeGooSignal 本 PR 只读对照，无仓内改动

## 背景

单标的宏观日批已在 #133 / #134 走 `POST /v1/skills/run`（`diagnose_macro`）。池级合成仍经 `synthesize_jev_with_retries` → `chat_diagnose` Agent Loop，再 `PUT .../pools/{workflow}`。需要固定池流水线 skill，JEV 方向仍由 GeeGooData Decision Choice 决定，LLM 仅根据 Choice 结果与标的摘要写中文理由。

## 已确认决策

| 项 | 选择 |
|----|------|
| skill 名 | `diagnose_macro_pool` |
| 路由 | 复用 `POST /v1/skills/run` |
| 查表 / PUT | skill **不算** canonical、**不**自动 PUT；workflow 组包后调用 skill，由 `MacroJevResultsWriter` 落库 |
| 标的 JEV | 仍由既有 `diagnose_macro` + GeeGooData 四路 JEV；池 skill **不**重跑标的 |
| 池方向 | GeeGooData `Choice`（同 unified 三选项），禁止 LLM 计票 / 固定权重捏造方向 |
| 中文理由 | Decision `chat`，基于 Choice + 标的摘要，不外露英文 CoT |
| overall | **保持** `chat_diagnose` + `synthesize_jev_with_retries`（本 PR 不重构） |
| detail_ref | `diagnose_macro_pool` |
| 方向口径 | skill `result.jev_direction` 为 ledger：`long` / `short` / `neutral`（bull/bear/unclear 仅在内部 Choice 映射） |
| confidence | skill 真实值；显式冲突时 cap ≤ 0.75；失败不得映射 neutral 冒充 success |
| Agent Loop | 池分支禁止 silent fallback；`--recover-session-id` 池路径弃用 |

## 请求 / 响应（最简）

**请求**

```json
{
  "skill": "diagnose_macro_pool",
  "macro_pool": {
    "date": "2026-11-20",
    "pool_workflow": "crypto",
    "source_batch_id": "macro-batch-run-id",
    "title": "加密",
    "expected_symbols": ["BTC-USD.CRYPTO", "ETH-USD.CRYPTO"],
    "symbols": [
      {
        "symbol": "BTC-USD.CRYPTO",
        "result_id": "sym:2026-11-20:crypto:BTC-USD.CRYPTO:...",
        "jev_direction": "long",
        "jev_reason": "…",
        "confidence": 0.82
      }
    ]
  }
}
```

**成功响应 `result`（精简，无重复 jev_unified）**

```json
{
  "date": "2026-11-20",
  "pool_workflow": "crypto",
  "source_batch_id": "macro-batch-run-id",
  "detail_ref": "diagnose_macro_pool",
  "jev_direction": "long",
  "jev_reason": "简体中文池级结论…",
  "confidence": 0.78,
  "conflict": false,
  "choices": [
    {"label": "看多", "probability": 0.78},
    {"label": "看空", "probability": 0.12},
    {"label": "不明确", "probability": 0.10}
  ],
  "input_refs": [
    {"symbol": "BTC-USD.CRYPTO", "result_id": "sym:…"},
    {"symbol": "ETH-USD.CRYPTO", "result_id": "sym:…"}
  ]
}
```

## 校验（fail-closed）

workflow 组包时必须满足：

- `date` ISO 日；`pool_workflow` ∈ 七池流水线（含 `bond`）；`source_batch_id` 非空
- `symbols` 非空、symbol 唯一；每条含 `result_id`、`jev_direction`∈{long,short,neutral}、`jev_reason` 达最小长度
- 禁止混池、混日、混 `source_batch_id`、重复 symbol、缺 result_id
- skill 内二次校验；不完整则 `status=failed`，不写 neutral 冒充成功

## 字段精简说明

| 保留 | 作用 |
|------|------|
| `jev_direction` | 唯一落库方向口径（long/short/neutral） |
| `jev_reason` | 池级中文结论 |
| `choices` | 当次 Choice 三档概率（与 `jev_unified.choices` 同形：`label` 为看多/看空/不明确） |
| `confidence` / `conflict` | 落库与 UI |
| `symbols[]` | 追溯 result_id + 最小依据摘要（非完整 lab_episode） |
| `detail_ref` | 与 Signal / Monday 契约对齐 |

删除 / 不发送：每条重复 `status`、`source_batch_id`、`lab_episode_summary`、嵌套 `jev_unified`、完整 OHLC、raw chat body。

## 联调阻碍（记录）

- 生产 `GET /macro/jev/results/symbols` 与 by-id 曾 HTTP 200 空 body；本 skill **不**依赖 stealth Mongo / 空 GET
- PUT 响应空体时不盲目重试；沿用 `MacroJevResultsWriter` 幂等语义

## 验收

- Go：`skills_run` 路由与池输入校验；Choice+理由 mock 集成；冲突 cap
- Python：齐套/缺失/混批次校验；池批无 `chat_diagnose`；`put_pool` 使用真实 confidence
- Fixture：`BTC-USD.CRYPTO` / `ETH-USD.CRYPTO`，批次日 `2026-11-20`（不调生产、不调真实模型）