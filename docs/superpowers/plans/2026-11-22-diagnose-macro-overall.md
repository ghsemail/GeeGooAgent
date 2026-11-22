# diagnose_macro_overall 实施计划

## 分支

| 仓库 | 分支 |
|------|------|
| GeeGooAgent | `cursor/macro-overall-jev-skill-1cb6` |
| GeeGooSignal | `cursor/macro-overall-jev-choices-1cb6` |

## 步骤

1. **Signal**：`ExpectedPoolMacroDate` + overall 输入日期校验；`choices` 写入 overall 文档；回归测试（错峰接受、陈旧拒绝、choices 往返）。
2. **Agent**：`lab_macro_overall_jev`（Choice + Chat，无 conflict cap）；`diagnoseskill.RunOverall` + 校验；`skills/run` 接线；`loader` 注册。
3. **Workflow**：`macro_overall_diagnose_skill.py`；`run_overall_batch` 改 skill；`put_overall_result` 传 `confidence`/`choices`/`detail_ref`。
4. **测试**：Go 单测 + Python 单测；单池 crypto 隔离 E2E（有 JEV 则真实调用并记时长，否则 mock 并标注阻塞）。

## 验证命令

```bash
# Agent
cd /agent/repos/GeeGooAgent
go test ./internal/diagnoseskill/... ./internal/workflow/strategydev/... -run 'MacroOverall|Overall' -count=1
python3 -m unittest scripts.ops.workflow.macro_overall_diagnose_skill_test -v

# Signal
cd /agent/repos/GeeGooSignal
go test ./internal/macro/... -run 'Overall|Choices|Orchestration' -count=1
```

## 单池材料（编排日 2026-11-22）

- `crypto` / `short` / BTC+ETH 空头摘要（用户提供的 canonical 池理由摘录）。