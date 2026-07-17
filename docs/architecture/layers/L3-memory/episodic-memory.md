# Episodic Memory（跨日摘要）

`recall_yesterday_summary` Tool 读取昨日盘前摘要：

1. **本地**：`workspace/reports/<YYYY-MM-DD>/<code>-premarket.md`
2. **Fallback**：`get_stock_daily_reports`（MCP）取同日 `pre_market` 内容

无上述数据时返回 `StatusSkip`（`implemented: true`），非未实现 stub。

详见 [tools-status.md](../L2-tools/tools-status.md#三decision决策辅助)。
