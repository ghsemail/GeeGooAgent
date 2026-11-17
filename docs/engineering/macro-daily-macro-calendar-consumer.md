# daily_macro 交易日历消费（Agent 侧，非 Bot 定稿）

App/Bot 扩仓由用户另行授权；本文档描述 **GeeGooAgent 已实现的发送/解析**、**终审要求的回显校验**、以及 **待 App 契约字段**（名称未定稿，不猜字段名写入生产解析）。

## 契约版本

| 层 | 常量 | 说明 |
|---|---|---|
| Agent 消费校验 | `mcp.CheckTradingDayConsumerContractVersion` | 回显 `date`/`market`/`code` 规则；与 App OpenAPI 版本独立对齐 |
| lab-detail 缓存 | `strategydev.LabDetailDailyMacroCacheContract` | JSON 输出契约；日历闭环落地后可能再 bump |

普通 signal-diagnose lab-detail（非 `daily_macro`）**不**要求 `date` 查询参数，**不**走 `RequireTradingDayChecker`。

## 现有 HTTP 消费（已实现）

**请求** `POST /checkTradingDay`（GeeGooBot mcp-api）：

```json
{
  "mcp_token": "<user>",
  "code": "<symbol>",
  "date": "YYYY-MM-DD"
}
```

- 实现：`internal/clients/mcp/methods.go` → `CheckTradingDayOnDate`
- **现行 Bot**：仅绑定 `mcp_token`+`code`；`date` 被忽略（Agent 仍发送，供未来 App）

**响应 `data`（已实现解析）** `internal/clients/mcp/types.go` → `TradingDayData`：

| 字段 | 类型 | Agent 用途 |
|---|---|---|
| `is_trading_day` | bool | bar 是否可作 macro cutoff 候选 |
| `date` | string | **必须**与请求的 `date` 一致，否则 fail-closed |
| `market` | string | **必须**非空且与 code 后缀一致（`.CRYPTO` 不得 echo `US`/`CN`/`HK`） |
| `code` | string | 若存在须与请求一致 |

校验：`mcp.ValidateCheckTradingDayData`；`MCPMacroTradingDayChecker` 在返回 `is_trading_day` 前执行。

## 待 App 正式契约（消费需求，字段名 **未定**）

以下为实现 daily_macro **真实日历** 前 Agent 需要能消费的能力（名称以 App OpenAPI 为准）：

1. **按查询日落库**：请求 `date` 参与服务端判定，响应 `date` 回显同一日历日。
2. **市场与时区**：响应能区分 `CRYPTO` 与 `US`/`HK`/`CN`；最好能携带 **IANA timezone** 或等价 `calendar_timezone`（避免 CRYPTO 复用美股 16ET）。
3. **半日市 / 真实 session close**：能提供 **当日有效收盘时刻**（exchange-local 或 RFC3339），供 macro cutoff 取代固定 `macroSessionSpec` 16:00/15:00；固定 16ET **不能**冒充半日收盘。
4. **CRYPTO（选项 C）**：日历日 = 供应商日 K 标签 `YYYY-MM-DD` 原文；**不用** UTC/上海 0:00 切日；**不用** UTC 23:59 作为 session 完成门闩；完成日以供应商已落库的完整日 K（close>0）为准；`checkTradingDay.date` 传该标签日。
4. **未知市场**：与现 Bot 一致返回 4xx 时，Agent 保持 fail-closed（不降级为「全日完成」）。

未落地前：生产连旧 Bot 时，校验器会对「今天回显」响应报错 → **daily_macro 监控 fail-closed**（预期行为）。

## 测试计划（已实现 / 待 App 联调）

| 场景 | 层级 | 状态 |
|---|---|---|
| 历史交易日：query=休市日前一交易日，echo 一致，`is_trading_day=true` | fixture + `ValidateCheckTradingDayData` | 已实现 OK fixture |
| 历史休市日：echo 一致，`is_trading_day=false` | fixture | 可增 fixture |
| 旧 Bot：query≠`data.date`（ today 回显） | `legacy_today_mismatch.json` | 已实现 fail |
| 无 `data.date` | `missing_date_echo.json` | 已实现 fail |
| `.CRYPTO` + `market=US` | unit test | 已实现 fail |
| HTTP lab-detail + httptest MCP | `signal_diagnose_lab_daily_macro_http_test.go` | 需按场景扩展 fixture 表 |
| CRYPTO session ≠ US 16ET close | `macro_monitor_session_test` | 已有 UTC 23:59 vs NY 16:00 分轨 |
| 半日：session close 来自 App 字段 | — | **阻塞 App**；禁止用固定 16ET 冒充 |

跨仓可注入测试：httptest MCP 读 `testdata/trading_day/*.json` 信封，不对 Bot 仓库开 PR。