# 个股盘前报告 — {{stock_name}} ({{code}})

**生成时间**: {{timestamp}}
**市场**: {{market}}
**bot**: {{bot_name}} ({{bot_type}}, id={{bot_id}})
**report_id**: {{report_id}}
**result**: {{result}}
**suggestion**: {{suggestion}}
**confidence**: {{confidence}}

---

## 一、市场背景（引用市场盘前）

> 来自当日 `pre_market` 全局报告（`get_market_pre_market_report`）。勿重复拉取三市场指数。

{{market_report_summary}}

---

## 二、个股新闻

{{stock_news}}

---

## 三、资金流向与分布

### 主力资金（日）
{{capital_flow}}

### 资金分布
{{capital_distribution}}

禁止输出 raw JSON；需有定量结论（如「主力净流入 X 亿」）。

---

## 四、周线技术分析

> `getMCPAnalysis` `period=weekly` **不返回 RSI/MACD**。无数据填「暂无」，勿编造。

### 价格概况
{{weekly_price_summary}}

### 关键支撑与阻力

| 类型 | 位置 | 重要程度 |
|------|------|---------|
| 短期阻力 | {{weekly_resistance_short}} | ★★★ |
| 中期阻力 | {{weekly_resistance_mid}} | ★★★ |
| 短期支撑 | {{weekly_support_short}} | ★★★ |
| 中期支撑 | {{weekly_support_mid}} | ★★★ |

### 趋势判断

| 周期 | 判断 | 说明 |
|------|------|------|
| 短期（1-5日） | {{weekly_short_term}} | {{weekly_short_reason}} |
| 中期（1-4周） | {{weekly_mid_term}} | {{weekly_mid_reason}} |

### 操作建议
{{weekly_trading_suggestion}}

---

## 五、Bot 盘前态度

上一交易日态度（服务端回溯最近 7 天）：

{{bot_attitude}}

---

## 六、综合预判

**reason**（≥80 字，引用具体数据）:

{{reason}}

```json
{
  "result": "long|short|neutral",
  "confidence": "high|medium|low",
  "suggestion": "buy|sell|hold"
}
```

---

## 七、操作建议

### 今日重点关注
{{key_watch_points}}

### 风险提示
{{risk_warnings}}

---

**报告生成人**: geegoo-agent · skill `pre_market_stock`
**关联市场报告**: {{market_pre_market_report_id}}
