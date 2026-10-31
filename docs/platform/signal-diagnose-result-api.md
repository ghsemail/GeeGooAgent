# Signal diagnose result API（Monday 联调）

SSOT：Chat 会话 metadata 内的 `SignalDiagnoseResult`（probe/diagnose 工具成功后服务端自动 Persist；runner 路径在 turn 结束时 Finalize）。

## 请求

- **Method / path**：`POST /v1/signal-diagnose/result`
- **Auth**：与 chat / lab-detail 相同（`X-MCP-Token` 或 body `mcp_token`）
- **Body JSON**：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `session_id` | string | 是 | Chat 会话 id |
| `run_id` | string | 否 | 省略时返回该会话 **最新** 一条 result；指定时返回匹配 `run_id` 且 `updated_at` 最新的一条 |
| `mcp_token` | string | 否 | 可与 Header 二选一 |

## 响应 JSON（Monday PR #57 映射）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `chart_probe` | object | 含 `bars`、`buy_merged`、`sell_merged`、episodes / price 等（与旧 ui-payload 中 chart_probe 同形） |
| `signal_eval` | object | Episode 评价（`method`、`buy_details`、`sell_details` 等） |
| `eval_judgment` | string | ComposeLLM / diag_compose Markdown（一～四段）；无则省略 |
| `lab` | object | Lab 摘要：有 `lab_episode_summaries` 时为 **detail_ref → summary** 的 map；仅 lab 元数据时为 `{ lab_episode_summaries?, lab_flow_price_summary?, lab_default_detail_ref? }` |

HTTP 404：会话不存在、无 result 或已过期（48h TTL）。

## 非目标

- 主路径 **不** 再写 `signal_diagnose_ui_payloads`；`GET/POST /v1/signal-diagnose/ui-payload` 仅 legacy 读已存条目。
- Skill 流程 **不依赖** `emit_workflow_card` 交付仪表盘数据。
