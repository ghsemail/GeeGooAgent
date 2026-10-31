# Signal diagnose result API（Monday 联调）

SSOT：Chat 会话 metadata 内的 `SignalDiagnoseResult`（probe/diagnose 工具成功后服务端自动 Persist；runner 路径在 turn 结束时 Finalize）。

Skill stepwise 在 probe 成功后会 Persist `chart_probe` / `signal_eval`，并在 probe 含 bars 时 **始终** 带上 **`lab`**（`lab_episode_summaries` 与/或 `lab_default_detail_ref` 壳，与 runner lab 摘要逻辑对齐）。**不再** 依赖 `skill_options.signal_diagnose.lab_mode` / sticky / inbound 的 `lab_mode` 决定是否写 `Result.lab`（Monday 侧 lab 面板 UX 仍由客户端自行控制；JEV 条读 `Result.lab`）。

## 请求

- **Method / path**：`POST /v1/signal-diagnose/result`
- **Auth**：与 chat / lab-detail 相同（`X-MCP-Token` 或 body `mcp_token`）
- **Body JSON**：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `session_id` | string | 是 | Chat 会话 id |
| `run_id` | string | 否 | 省略时返回该会话 **最新** 一条 result。指定时优先匹配 Persist 条目上的 `run_id` / `result_key`；若无匹配且会话内仅有一条未过期 result，则返回该条（兼容 Monday 侧自有 run 文档 id）。Skill stepwise 路径建议在首次响应后改用 Persist 内的 `run_id`（`flow-…`）或省略本字段 |
| `mcp_token` | string | 否 | 可与 Header 二选一 |

## 响应 JSON（Monday PR #57 映射）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `chart_probe` | object | 含 `bars`、`buy_merged`、`sell_merged`、episodes / price 等（与旧 ui-payload 中 chart_probe 同形） |
| `signal_eval` | object | Episode 评价（`method`、`buy_details`、`sell_details` 等） |
| `eval_judgment` | string | ComposeLLM / diag_compose Markdown（一～四段）；无则省略 |
| `lab` | object | Lab 摘要：有 `lab_episode_summaries` 时为 **detail_ref → summary** 的 map；仅 lab 元数据时为 `{ lab_episode_summaries?, lab_flow_price_summary?, lab_default_detail_ref? }` |
| `trade_direction` | string | 可选：`both`（默认）或 `long_only`。`long_only` 时 **`lab` / Persist 的 `lab_episode_summaries` 不含 `sell:*`**（与 `signal_eval` 一致，不做做空 Episode 列表）；probe 原始 `sell_merged` 仍可在 `chart_probe` 中表示平多卖出信号 |

HTTP 404：会话不存在、无 result 或已过期（48h TTL）。

## 非目标

- 主路径 **不** 再写 `signal_diagnose_ui_payloads`；`GET/POST /v1/signal-diagnose/ui-payload` 仅 legacy 读已存条目。
- Skill 流程 **不依赖** `emit_workflow_card` 交付仪表盘数据。
