# diagnose_macro_overall 设计（已批准）

## 目标

用 HTTP skill 替代宏观整体 `overall` 的 Agent chat 合成：纯计算、不 GET/PUT canonical；编排/workflow 负责 ledger、七池 `result_id` 与 macro-api `PUT`。

## 请求契约（禁止扩展字段）

`POST /v1/skills/run`

```json
{
  "skill": "diagnose_macro_overall",
  "macro_overall": {
    "date": "YYYY-MM-DD",
    "pools": [
      {
        "pool_workflow": "crypto",
        "jev_direction": "short",
        "jev_reason": "池诊断理由"
      }
    ]
  }
}
```

- 顶层 `date`：编排日（上海日历），响应原样返回。
- 每池仅 `pool_workflow`、`jev_direction`（`long|short|neutral`）、`jev_reason`。
- 无 `choices`/`confidence`/`title`/`result_id`/`source_batch_id` 等输入。

## 成功响应 envelope

沿用现有 skills/run 外形：

- `status`: `"ok"`
- `session_id`
- `skill`: `"diagnose_macro_overall"`
- `result`：`date`、`detail_ref`=`diagnose_macro_overall`、`jev_direction`、`jev_reason`、`confidence`、`choices`（`[{label:"看多|看空|不明确", probability}]`）
- 不返回 `input_refs`、`conflict`
- 失败：`status: failed` + `error`，可带 `session_id`；不伪造 `neutral`/概率

## 计算语义

1. **Choice**（GeeGooData `/v1/decision/choice`）：state = 各池 workflow 标签 + 池级 `jev_direction`/`jev_reason` 摘要；宏观 **long/short = 风险偏好 risk-on / risk-off**；债券/波动/商品货币方向不得机械等同于股指。
2. **理由 Chat**（`/v1/decision/chat`）：基于原始 Choice + 概率与各池结论写中文 Markdown（`## 整体结论`），禁止英文 CoT、固定权重、多数票。
3. **不**沿用池内 `long+short` 时的 confidence cap 0.75；**不**输出 `conflict` 字段。

## 池列表校验（skill 侧）

- 合法 `pool_workflow`：`cn_a`/`hk`/`us`/`commodity_fx`/`bond`/`crypto`/`volatility`
- 数量 **1–7**，无重复；空/非法/重复 → 失败
- **无**测试开关或额外请求参数
- 部分池运行：结论与理由须明示仅覆盖已提供池，不声称七池全资产

## 与生产落库边界

- **Skill**：1–7 池均可调用（含单池 crypto 验证）。
- **Signal `PUT .../overall` success**：仍须 **七池齐套** + 冻结池 `result_id`；部分池 skill 结果 **不得** 冒充正式 overall success。
- Workflow：`overall` 从 chat 改为 skill；向 macro-api 传递真实 `confidence`/`choices`/`detail_ref`，不再硬编码 `0.6`、不从正文解析方向。

## Signal 配套（GeeGooSignal）

- **choices**：overall 层完整持久化 + GET 返回（与 pool 层对齐）。
- **日期**：整体 `date` ≠ 各池 macro `date`（错峰）；`validateAndFreezeOverallInputs` 用 `ExpectedPoolMacroDate(编排日, pool)` 校验池快照日期并冻存 `inputs[].date`，禁止删掉校验后任意旧池可用。