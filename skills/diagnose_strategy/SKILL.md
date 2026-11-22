---
name: diagnose_strategy
description: 策略诊断四路事实 + JEV（macro-api :3220，非 Agent Loop）
version: "1.0.0"
---

# diagnose_strategy

固定流水线 HTTP skill：拉取 **GeeGooSignal macro-api** `:3220` 四路 diagnose 事实 → 串行四路 **JEV 决策**（GeeGooData Choice）→ 一次 **jev_unified**（冲突 confidence ≤ 0.75）。LLM 仅撰写各路与总路的 `recommended_reason`，不改 direction。

**不是** `signal_diagnose` Agent Loop，**不是** 诊断 Episode SSE。

## 触发

`POST /v1/skills/run`

```json
{
  "skill": "diagnose_strategy",
  "strategy": {
    "code": "00700.HK",
    "signal_frequency": "60m",
    "as_of": "可选"
  }
}
```

- 必填 `strategy.code`、`strategy.signal_frequency`
- `as_of` 可选；省略时与 3220 一致（最新已完成 K 线）
- 鉴权与 `mcp_token` 字段沿用现有 skills/run

## 3220 路径

- `POST /v1/diagnose/strategy/kline-pattern`
- `POST /v1/diagnose/strategy/price-analysis`
- `POST /v1/diagnose/strategy/technical-analysis`
- `POST /v1/diagnose/strategy/index-trend`

## 回包

外壳：`status`、`session_id`、`skill`；完整载荷在 `result`（四路事实数组原样 + 四路 JEV + `jev_unified`）。`direction` 仅为 `bull` / `bear` / `unclear`；仅 `jev_unified` 含 `confidence`、`conflict`。

实现：`internal/diagnoseskill`、`internal/runtimeapi/skills_run.go`。