# 宏观监控池日批（asia / us / crypto）

## 职责

按监控池对每标的跑 `signal_diagnose`（Lab JEV），再写入 GeeGooSignal **macro-api**：

1. `POST /macro/decision/inline/ingest`
2. `POST /macro/batch/runs`

契约详见 GeeGooSignal `docs/MACRO_LAB_BATCH.md`。

## 当前线上缺口（2026-11-08 核查）

日批 **能跑通**（summary 如 `asia ok=9/9; ingest upserted=9`），但 `pools[].items[]` **普遍缺少**：

| 字段 | 影响 |
|------|------|
| `agent_session_id` / `agent_run_id` | macro-api 无法自动 `lab-detail` enrich；运营台 prefetch 只能靠 Signal 诊断 log |
| `jev_reason`（batch item 上） | 日批日志详情无 compose 长文 |

**编排脚本未纳入本仓库 Git**（全库无 `ingest upserted` 源码）；请在实际触发方（本机 Automation / 外部 runner）按下列契约补齐字段。

## 每个成功标的必须写入

**inline ingest item** 与 **batch item** 保持一致：

```json
{
  "code": "000001.SH",
  "jev_direction": "short",
  "jev_reason": "<compose 长文>",
  "detail_ref": "buy:239",
  "agent_session_id": "chat-…",
  "agent_run_id": "flow-…"
}
```

推荐在同一轮诊断结束后：

1. `POST /v1/signal-diagnose/result` 取 `run_id` + `lab` 默认 `detail_ref`
2. 可选 `POST /saveSignalDiagnoseLog`（运营台按标的 prefetch 兜底）
3. 再 POST macro ingest + batch

## crypto 与 Agent PG

日批若报 `GEEGOO_PG_DSN is not connected`，在 Agent 主机执行：

`python scripts/ops/fix/_fix_pg_session_remote.py`

并确认 `agent-runtime` 健康 `:3400/health`。

## 回补历史四维

仅当 Signal 有该标的 `flow-*` 或 batch 已带 agent ids：

```bash
python scripts/macro_backfill_lab_episode_summary.py --date YYYY-MM-DD
```

## 运营台展示

- 合并 inline 使用 query **`date=`**（非 `decision_date`），见 `MacroServer.getQuotesSnapshot`
- 行上需 `jev_batch_date` + 二次 dated GET 才会带出 `lab_episode_summary`
