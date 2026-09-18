---
name: param_tune
description: 策略参数调优 Workflow（规划中）：串行尝试 months_back / frequency 等参数组合，改善买卖点可见性。
---

# 策略参数调优 Workflow

## 触发

- **Chat**：买卖点不明显、信号太少、帮我调参
- **Cron**：`jobs.json` 中配置 `skill=param_tune`（规划中）

## 阶段

1. `resolve_context` — 解析标的与基准策略
2. `baseline_probe` — 默认参数 probe
3. `pick_param_variants` — 生成参数组合
4. `probe_foreach` — 串行 probe
5. `summarize` — 对比各组买卖点

## 状态

`planned` — Chat 检测已接入，完整 runner 待实现。
