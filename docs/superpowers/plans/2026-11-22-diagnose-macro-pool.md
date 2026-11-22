# diagnose_macro_pool 实施计划

> **Goal:** 池 JEV 合成走 HTTP skill，workflow 负责 PUT；overall 仍 Agent Loop。

**Architecture:** `internal/diagnoseskill` 新增 `RunPool` + `strategydev.ResolveMacroPoolJEV`（Choice + 中文 reason）；`runtimeapi` 注册 `diagnose_macro_pool`；Python `macro_pool_diagnose_skill.py` 组包/解析；`run_macro_monitor_lab_batch` 池路径改调 skill。

## Task 1: Go pool JEV + tests

**Files:** `lab_macro_pool_jev.go`, `lab_macro_pool_jev_reason.go`, `pool.go`, `pool_test.go`, `diagnose_skill.go`, `skills_run.go`, `loader.go`, `skills/diagnose_macro_pool/SKILL.md`

- [ ] Choice + reason 单测（mock JEV server）
- [ ] 输入校验单测（缺标的、重复、非法方向）

## Task 2: Python workflow + client

**Files:** `macro_pool_diagnose_skill.py`, `macro_pool_diagnose_skill_test.py`, `run_macro_monitor_lab_batch.py`, `macro_jev_results_client.py`, `macro_jev_backfill_pools_overall.py`, `macro_lab_batch_health.py`

- [ ] `synthesize_pool_jev_with_skill_retries` 替换池 `synthesize_jev_with_retries`
- [ ] `put_pool_from_synthesis` 使用 `pool_jev.confidence`
- [ ] backfill 池段走 skill；overall 仍 chat

## Task 3: 验证

- [ ] `go test ./internal/diagnoseskill/... ./internal/runtimeapi/...`
- [ ] `python3 -m unittest` 相关 workflow 测试
- [ ] `run_macro_jev_integration_tests.sh`（如适用）