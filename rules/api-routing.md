# geegoo mcp API 路由

## 统一服务

| 服务 | 端口 | Base URL | api_key 前缀 | 用途 |
|------|------|----------|--------------|------|
| **geegoo mcp** | 5700 | `config.geegoo_url`（与 `base_url` 同值） | `sk-...` | Bot CRUD、行情、workflow、报告、技术分析、策略 |

原 geegoo mcp（5700）与 `mk-` API Key 已废弃；全部接口合并至 geegoo mcp（5700）。

`mcp_token` 从配置/环境读取，**禁止硬编码**。

## 按场景选接口

| 场景 | 接口 |
|------|------|
| 执行盘前 workflow | `checkTradingDay` → `getReportBotCodes` → … |
| 按日期查盘前/盘中/盘后报告 | `getStockDailyReports` |
| 指数/个股技术分析 | `getMCPAnalysis` |
| 创建盘前报告 | `createPreMarketReport` |
| 查询盘前报告列表 | `getPreMarketReports`（ObjectId 序列化 bug 已于 2026-05-20 修复） |

## 盘前资金相关（同时调用）

| API | 参数 | 说明 |
|-----|------|------|
| `getCapitalFlow` | `period=DAY` | 资金流向 |
| `getCapitalDistribution` | 仅需 `code` | T-1 资金分布 |

## getMCPAnalysis period 值

| 用途 | 正确值 | 错误值 |
|------|--------|--------|
| 小时级（指数） | `hourly` | `hour` |
| 周线（个股 S/R） | `weekly` | — |
| 日线 | `daily` | — |

响应字段名为 **`analysis_result`**，不是 `analysis`。

## HTTP 调用规范

- Header：`Authorization: Bearer <sk-api-key>`，`Content-Type: application/json`
- `checkTradingDay` body：`{"mcp_token": "...", "code": "00700.HK"}`（无需 `date`）
- 超时建议 ≥ 60s；首次连接失败可重试
