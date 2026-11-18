# 宏观监控池日批（七池 + overall）

## 职责

按监控池对每标的跑 `signal_diagnose`（Lab JEV），再写入 GeeGooSignal **macro-api**：

1. `POST /macro/decision/inline/ingest`
2. `POST /macro/batch/runs`
3. **canonical 真源**：`PUT /macro/jev/results/symbols` → `.../pools/{workflow}` → `.../overall`（见 GeeGooSignal `docs/MACRO_JEV_RESULTS_API.md`）

分时段 crontab 与日账本见 `docs/MACRO_JEV_LEDGER_AND_REPLAY.md`。集成测试：`bash scripts/ops/workflow/run_macro_jev_integration_tests.sh`。

## 工作日时刻表（Asia/Shanghai，仅周一–周五）

| 时刻 | cron `--workflow` | 说明 |
|------|-------------------|------|
| 05:30 | `us` | 晨间块 |
| 05:32 | `commodity_fx` | 晨间块（日期规则见下「推断」） |
| 05:35 | `bond_us` | 美债分段；与 `bond_sh` 合并为 **单一** `bond` 池 |
| 05:38 | `crypto` | 晨间块 |
| 05:40 | `volatility` | 晨间块（日期规则见下「推断」） |
| 15:30 | `cn_a` | 午后块 |
| 16:00 | `bond_sh` | 上交所债分段 |
| 16:30 | `hk` | 午后块 |
| 17:00 | `overall` | 读本日账本七池后合成 |

**旧生产时刻（仅供 diff）**：`cn_a` 07:00、`hk` 07:02、`volatility` 07:10、`bond_sh` 08:55、`overall` 09:10、crypto `America/New_York` 16:15（其余晨间时刻与上表前半相同）。

安装：`bash scripts/ops/workflow/install_macro_jev_crontab.sh`（**不**在本 PR 自动改生产 crontab）。

## 分析日 / pipeline 日期（上海日历）

- **编排窗口（账本文件名、`--date auto` 的 orchestration 日）**：当日 **Asia/Shanghai 日历日** `D`。
- **macro-api `date` 入库**：
  - **产品明确确认**：`us`、`crypto` → **上一美股交易日**（相对 `D`，周末回退；**不含**美国联邦假日历）。
  - **产品明确确认**：`cn_a`、`hk` → **同日历日 `D`**。
  - **由晨/午分组推断（文档标注为推断，非口头逐名确认）**：`commodity_fx`、`volatility`、`bond_us` 与 `us`/`crypto` 同属晨间块 → 上一美股交易日；`bond_sh` 与 A 股同属午后块 → 日历日 `D`。
  - **合并 `bond` 池级** canonical：编排日 `D`；标的来自 `bond_us` / `bond_sh` 两日 segment 合并。

实现：`macro_jev_orchestration.resolve_run_dates` / `expected_pool_macro_date`。

## 失败重试

1. **单标的**：批内诊断失败时，进程内最多 3 次、间隔 `MACRO_BATCH_RETRY_DELAY_SEC`（默认 **600**，即 10 分钟）。见 `diagnose_symbol_with_retries`。
2. **`overall` 17:00**：若账本中任七池尚无 `status=success`，**先**按 `POOL_RERUN_CRON_WORKFLOWS` 重跑对应 cron（`bond` 重跑 `bond_us`+`bond_sh` 并 finalize），再读本日账本做整体合成；**不**跳过缺池、**不**伪造 success。

## 夏令时（DST）说明

尚未单独编排 DST 专用 crontab。美东夏令时期间，美股/加密会话相对标准时的上海本地观感会 **早 1 小时**；本仓库 crontab 仍按 **Asia/Shanghai 固定钟点** 写入，**未**单独维护 DST 时刻表。

## Agent 主机定时任务

单池：`--workflow cn_a|hk|us|commodity_fx|bond|crypto|volatility`。债券分段：`bond_sh` / `bond_us`。宏观合成：`--workflow overall`。

日期：`--date auto`（编排日 = 上海当日）；macro 入库日由 workflow 自动映射（见上）。

**过渡期**：`--workflow all` 仍可本地使用。

`agent.env` 需含 `MACRO_API_KEY`。

## 本地集成

```bash
bash scripts/ops/workflow/run_macro_jev_integration_tests.sh
```