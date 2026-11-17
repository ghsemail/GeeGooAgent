# 宏观 JEV · 分时段 crontab 账本与历史回放调查

## 生产编排入口（运维事实）

- Agent 宏观日批入口：**主机 crontab** → `scripts/ops/workflow/run_macro_monitor_lab_batch.py`（与 geegoo scheduler 分离）。
- 安装脚本：`scripts/ops/workflow/install_macro_jev_crontab.sh`（`CRON_TZ=Asia/Shanghai`，crypto 段 `America/New_York` 16:15）。
- **同一 pipeline `date`** 下多池 **独立 cron、独立 `batch/runs` run.id**；overall **09:10** 不得用 `GET ...?source=canonical` 按日捞「当前 latest」填缺失。

## 编排冻结来源（本 PR 行为）

| 层级 | 冻结来源 |
|------|----------|
| 标的 | 当次 cron 首条 `batch/runs` 的 **run.id** → `source_batch_id`；`batch_started_at` 同 run |
| 池 | 同池当次 finalize run.id；`inputs[].result_id` 仅指向 **同日同池** 已 PUT 标的 |
| 整体 | 仅读 `~/.geegoo/macro_jev_pool_ledger_{date}.json`（`MACRO_JEV_LEDGER_DIR` 可覆盖）中 **七池 result_id**，再 `GET by-id` 校验同日、tier、ledger 记录 status；**禁止** pipeline/auto 拼池 |

账本每条含：`result_id`、`source_batch_id`、`batch_started_at`、`status`、`cron_workflow`（如 `bond_sh` vs `bond_finalize`）。

### bond 双分段

- `bond_us` / `bond_sh`：各自 run.id 写标的层；池层 **defer**。
- 合并：`try_finalize_bond_pool_jev` 用 **finalize 专属 run.id** + manifest `pools[].symbols` 全量；symbol `result_id` 来自分段 PUT（可 `by-id` 绑定，**不**用跨日旧 success）。

### overall 与 crypto 时序

- crontab：**09:10 overall** 早于当日 **16:15 ET crypto**（华东 pipeline 日对齐规则不变）。
- 故 09:10 若账本缺 `crypto`：**预期** overall 写 **failed/incomplete**（新 `source_batch_id`），**不得**保留昨日 success 为 current。

## 2025-11-17 生产只读审计（摘要）

七池 **7×run.id**、**33** 标的 success 可导入；**7×pool_jev 无 agent_run_id**；**overall 0**；canonical **0**。最小闭环见 **`docs/MACRO_JEV_BACKFILL_MINIMAL_LOOP.md`**（Phase A 标的 PUT + Phase B/C 共 8 次合成，不重跑 33 Lab）。分类器：`macro_jev_audit_schema.py`。

## 历史同源原始记录 · 三层审计回放（可行性，只读设计）

**现状（Signal main b3eb834 已上 macro-api，三表历史为 0）**

| 数据源 | 可回放内容 | 能否填满 canonical 三层 |
|--------|------------|-------------------------|
| `macro_batch_runs` + `pool_jev` | 标的 items、池/整体 **jev_direction/reason**、session/run（若有） | 标的/池/整体 **仅当字段完整**；无 agent ids 则 symbol 不能标 success |
| `macro/decision/inline` | 单标的 direction/reason/detail_ref/session | 标的层可部分回填 |
| 旧 GET `jev/pipeline` | 七池状态文案 | **不能**单独作为整体 PUT 依据（无七条 pool `result_id` 链） |

**结论**

1. **不编造**：无真实池/整体 **合成原文**（含 session/run + 足长 `jev_reason`）→ **不得** `status=success` PUT 池/整体。
2. **可审计重放（未来脚本，非本 PR 生产执行）**：只读拉取 batch/inline → 按契约校验 `item_jev_complete` / `synthesis_pool_jev_valid` → 顺序 PUT symbol → pool → overall；失败行写 failed/incomplete；`source_batch_id` 用 **原 run.id** 或打标 `replay-{run.id}` 避免污染新日批 latest（须运维授权与 Signal 约定）。
3. **不可从旧批「current success」冒充新日失败/缺失**；回放与 live 均须用 **账本或显式 run 绑定**。

## 生产最小验证（无全量 Lab）

1. macro-api 已 b3eb834；选一工作日、**单池** cron（如 `us`），看 exit 与日志 `jev-results PUT`。
2. `GET .../symbols?date=&pool_workflow=us&source=canonical` 有当次 run.id 对应行。
3. 当日七池跑完后查账本文件七键；再 overall cron：`GET .../overall?date=&source=canonical` 为 **当日新 run** 的 success 或 **显式 failed**（非空且非昨日 id）。
4. bond：核对分段标的 + finalize 池 `expected_manifest_source=batch_run`。

## 本地集成

```bash
bash scripts/ops/workflow/run_macro_jev_integration_tests.sh
```

需本机 `127.0.0.1:27017` Mongo；`SKIP_MACRO_JEV_INTEGRATION=1` 可仅跑单元测试。