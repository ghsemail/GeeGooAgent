# Attitude → Result 映射

## 枚举映射（强制）

| `get_bot_yesterday_attitude` | `create_pre_market_report.result` |
|------------------------------|-----------------------------------|
| `bullish` | `long` |
| `bearish` | `short` |
| `neutral` | `neutral` |

两套枚举**不可混用**。`attitude` 三态与 `result` 三态含义对应，但字段名与取值不同。

## 404 处理

`getBotYesterdayAttitude` 返回 HTTP 404 / `code=105`（昨日无 attitude 记录）时：

- 视为**正常空状态**，不是错误
- 默认 `attitude = neutral`，`analysis_report = ''`
- **不要重试**；确认 404 后立即继续后续步骤

## 周线趋势解析（勿用关键词误判）

`getMCPAnalysis` 周线报告中的「短期/中期/长期」结论段落为准，**禁止**全文搜索「多头」「空头」等词做判定。

| 趋势结论关键词 | 映射 |
|----------------|------|
| 看涨、强势看多、量价齐升 | bullish |
| 看跌、偏空、空头排列、下降通道 | bearish |
| 其余 | neutral |

优先以「短期」段落下的判断为准。
