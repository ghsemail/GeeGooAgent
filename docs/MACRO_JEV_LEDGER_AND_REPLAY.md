# 宏观 JEV · 分时段 crontab 账本与历史回放调查

## 生产编排入口（运维事实）

- Agent 宏观日批入口：**主机 crontab** → `scripts/ops/workflow/run_macro_monitor_lab_batch.py`（与 geegoo scheduler 分离）。
- 安装脚本：`scripts/ops/workflow/install_macro_jev_crontab.sh`（**Asia/Shanghai** 全表；crypto 已不再使用 `America/New_York` 独立行）。
- **同一编排日 `D`**（上海日历）下多池 **独立 cron、独立 `batch/runs` run.id**；`overall` **17:00** 仅读本日 `macro_jev_pool_ledger_{D}.json`，**禁止**用 `GET ...?source=canonical` 按日捞「当前 latest」填缺失。

### 工作日时刻（Asia/Shanghai，Mon–Fri）

| workflow | 时刻 |
|----------|------|
| us | 05:30 |
| commodity_fx | 05:32 |
| bond_us | 05:35 |
| crypto | 05:38 |
| volatility | 05:40 |
| cn_a | 15:30 |
| bond_sh | 16:00 |
| hk | 16:30 |
| overall | 17:00 |

旧表：overall 09:10、cn_a 07:00、hk 07:02、volatility 07:10、bond_sh 08:55、crypto 16:15 ET。

## 编排冻结来源

| 层级 | 冻结来源 |
|------|----------|
| 标的 | 当次 cron 首条 `batch/runs` 的 **run.id** → `source_batch_id` |
| 池 | 同池 finalize 的 run.id；`inputs[].result_id` 指向 **约定 macro `date`** 下已 PUT 标的 |
| 整体 | 仅读账本 **七池 result_id**；`GET by-id` 校验 tier、ledger status、**各池期望 macro `date`**（晨间池 = 上一美股交易日，午后池 = `D`） |

账本键：`~/.geegoo/macro_jev_pool_ledger_{D}.json`（`MACRO_JEV_LEDGER_DIR` 可覆盖）。**`D` = 编排窗口，非一律等于各池 macro `date`。**

### 分析日规则（摘要）

- **明确**：`us`/`crypto` → 上一美股交易日；`cn_a`/`hk` → 日历日 `D`。
- **推断（文档标注）**：`commodity_fx`、`volatility`、`bond_us` 与晨间块同锚；`bond_sh` 与午后块同锚；**合并 `bond` 池** tier 在 `D`。

### bond 双分段

- `bond_us` / `bond_sh`：各自 cron、各自 macro 日；池级 **defer** 至 `try_finalize_bond_pool_jev`（合并 segment 标的 + **编排日 `D`** pool PUT）。
- overall 缺 `bond`：重跑两段 finalize 后再合成。

### overall 17:00 与缺池

- 账本未满七池 success：`run_overall_batch` → `refresh_ledger_before_overall` 按池重跑 cron，**再**合成；失败写 blocked/failed，**不**用昨日 success 顶替。

### DST

未单独维护 DST crontab；夏令时期间美东会话在上海观感上早 1 小时，**本仓库仍用固定上海钟点**。

## 本地集成

```bash
bash scripts/ops/workflow/run_macro_jev_integration_tests.sh
```

需本机 `127.0.0.1:27017` Mongo；`SKIP_MACRO_JEV_INTEGRATION=1` 可仅跑单元测试。