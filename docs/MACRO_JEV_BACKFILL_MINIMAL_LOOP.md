# 宏观 JEV · 最小真实闭环回填方案（只读审计驱动）

> **本稿为方案与测试约束，非生产执行。** 现网 macro-api **b3eb834**，canonical 三表历史 **0**。

## 2025-11-17 只读生产审计（未回放 / 未推理）

| 观测 | 结论 |
|------|------|
| 七池各 **1** 个最新 finished `macro_batch_runs` | **7 个不同 run.id**（不可统一 `source_batch_id`） |
| 标的 **33** success | 方向、理由、`detail_ref`、`agent_session_id`、`agent_run_id` **齐全** → **可审计导入 symbol PUT** |
| `pool_jev` 七条均 success | **全部缺 `agent_run_id`**（多缺 session）→ **不可**把 legacy `pool_jev` 当 canonical success 直写 |
| overall finished 原文 **0** | 无整体合成可复用 → **须新一轮真实 overall 合成** |
| bond | **merged segments** 单 merged run；**禁止**伪拆成独立池来源或伪造分段 run |
| batch 标的 `lab_episode_summary` | 缺 → **可选**，不阻断 symbol 核心 JEV success |
| 2025-11-18 completed | **0**；canonical **0** |

**不能做的**

- 三层一次性「success 回放」 pretending 历史已齐。
- 七池共用一个 `source_batch_id`。
- 用旧 `pool_jev` 文本冒充带 session/run 的 canonical 池 success。
- 为 overall **编造** run.id / 无合成会话的整体 success。

## 最小真实闭环（推荐）

在运维授权、**非重叠日**、**batch 无活动** 前提下：

### Phase A · 标的层（无 Lab 重跑）

**输入**：只读导出各池 finished run（保留 **原 `id`→`source_batch_id`、原 `date`、`batch_started_at`**）。

**动作**：对 33 条通过 `macro_jev_audit_schema` 的标的，调用现有 `MacroJevResultsWriter.put_symbol_from_batch_item`（或等价 PUT），`revision=0` 终态 success（可选先 running@0 再 success@1，与线上一致）。

**成本**：33 次 HTTP PUT + 0 次 Lab 诊断。

**工具（待实现，本 PR 仅分类器 + 测试）**：

- `scripts/ops/workflow/macro_jev_backfill_symbols.py` — `--date`、`--from-export`、`--dry-run`（默认 dry-run）
- 依赖：同目录 `macro_jev_results_client.py`（cron 已 `cd` 脚本目录）

### Phase B · 池层（仅合成，不重跑标的）

**输入**：Phase A 写入的 symbol `result_id`（按池 freeze）。

**动作**：对每个 `pool_workflow`：

1. `POST /macro/batch/runs` **新心跳** → 新 `run.id` 作池层 `source_batch_id`（**每池独立**，与 Phase A 标的 run.id **不同**属正常）。
2. 调用现有 **`synthesize_jev_with_retries`**（与 `run_pool_jev_only` 相同），输入为 **Phase A canonical 或 batch items 摘要**，拿到 **新** `session_id` / `run_id`。
3. `PUT .../pools/{wf}`，`inputs[].result_id` 仅引用同日同池 success 标的；bond 用 **finalize merged manifest** 全量 expected（`try_finalize_bond_pool_jev` 语义）。

**成本**：**7 次** Agent 池级合成（chat），非 33 Lab。

**禁止**：复制 legacy `pool_jev` 无 `agent_run_id` 仍标 success。

### Phase C · 整体（仅合成）

**前置**：`macro_jev_pool_ledger_{date}.json` 七键 success（由各池 Phase B 写入 `record_pool_ledger`）。

**动作**：

1. `overall_inputs_from_ledger` + `pool_synth_rows_from_ledger`（**禁止** canonical GET latest 填洞）。
2. **一次** `synthesize_jev_with_retries`「宏观整体」。
3. 新 overall `batch/runs` → `PUT .../overall` 七条 pool `result_id`。

**成本**：**1 次** Agent 整体合成。

### bond 特别说明

- 标的 `source_batch_id` 可仍绑定 **segment** run.id（若 BSON 保留分段记录）；池层 **仅** merged finalize 的 run.id + manifest **HYG+TLT 全量**（与线上一致）。
- **不得**为分段各造池 success 再拼 overall。

## 与线上一致性

| 项 | 要求 |
|----|------|
| 多 cron | 七池 **七个** batch id；回填亦保持 **每池一 id**（标的可绑历史 run.id） |
| overall cron | 仅账本 + by-id；缺 crypto（09:10 早于 16:15 ET）→ **failed/incomplete** 为设计内行为 |
| 部署 | `cd /home/ubuntu/.geegoo/geegoo-agent`；**ff-only** `git pull`；同目录 Python import；**无**编译/安装/runtime 重启 |
| 运维 | 无活动 batch；**保留** tracked `.out`/`.pid` dirty；不 reset/stash |

## 代码现状（PR #126）

| 已有 | 备注 |
|------|------|
| 日批 PUT 接线 + ledger | — |
| `macro_jev_audit_schema.py` 分类 | — |
| `macro_jev_backfill_symbols.py` Phase A | 默认 dry-run；`--apply` + `--confirm-no-active-batch`；非 localhost 需 `--allow-production-host` |
| `macro_jev_backfill_pools_overall.py` B/C | `--recorded-synthesis-dir` 仅回放已录 synthesis（集成/演练）；生产 `--apply` 走真实 Agent 合成 |
| 集成测试 | `run_macro_jev_integration_tests.sh` 全开 HTTP + Go |

## 可行性结论

| 路径 | 判定 |
|------|------|
| 审计导入 33 标的 + 8 次合成（7 池 + 1 整体） | **可行**，与契约一致，**无需**重做 33 Lab |
| 无 Agent 的合成（纯手工拼 pool/overall 文本） | **不可行**（缺 session/run，且违反不造假） |
| 单日三层 success「纯回放」无合成 | **不可行**（11/17 pool/overall 身份不足） |

## 验证命令（开发机）

```bash
cd scripts/ops/workflow
python3 -m unittest macro_jev_audit_schema_test.py -v
bash run_macro_jev_integration_tests.sh
```

生产只读审计分类（示例 JSON 占位）：

```bash
python3 -c "
from macro_jev_audit_schema import classify_day_export
# 运维将 BSON/API 导出填入 pool_runs_by_workflow
"
```