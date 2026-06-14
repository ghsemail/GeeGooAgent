# 盘前报告格式与 API 校验

## create_pre_market_report 必填字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `mcp_token` | string | 用户令牌 |
| `code` | string | 标的代码 |
| `stock_name` | string | **字段名是 `stock_name`，不是 `name`** |
| `bot_id` | string | 来自 `get_report_bot_codes`，禁止留空 |
| `bot_name` | string | 来自 `get_report_bot_codes` |
| `bot_type` | string | 来自 `get_report_bot_codes` |
| `result` | string | `long` / `short` / `neutral` |
| `confidence` | string | `high` / `medium` / `low` |
| `reason` | string | 判定依据，非空 |
| `suggestion` | string | `buy` / `sell` / `hold` |
| `report` | string | 报告原文，非空 |

建议同时提供：`summary`、`support`、`resistance`。

## 报告模板（九章）

模板文件：`skills/pre_market/template.md`

1. 指数分析（道琼斯/纳斯达克/上证/深证/恒生）
2. 市场新闻摘要（US/CN/HK）
3. 个股新闻
4. 资金流向
5. 资金分布
6. 周线技术分析（均线/支撑/阻力/趋势/成交量/操作建议）
7. 昨日 Bot 态度
8. 综合预判
9. 操作建议

## 周线技术分析（API 实际字段）

`getMCPAnalysis` `period=weekly` **不返回 RSI/MACD**。模板第六节若含 RSI/MACD 占位符，填「暂无」或从其他段落推导，勿编造。

实际可用字段：支撑位、阻力位、均线位置、趋势判断、成交量信号、操作建议。

## 资金分布格式化

禁止写 raw JSON。推荐格式：

```text
超大单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
大单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
中单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
小单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
更新时间：YYYY-MM-DD HH:MM:SS
```

## 本地留档路径

```
{workspace_root}/reports/<YYYYMMDD>/<code>-premarket.md
```

## 飞书推送

完整 Markdown 存本地；推送仅发摘要（约 2000 字符限制）。港股个股新闻可无数据，不阻塞流程。
