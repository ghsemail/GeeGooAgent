# 盘前准备报告

**生成时间**: {{timestamp}}
**市场**: {{market}}
**report_id**: {{report_id}}
**result**: {{result}}
**suggestion**: {{suggestion}}
**confidence**: {{confidence}}

---

## 一、昨日美股走势

### 道琼斯工业指数 (DJI)
| 指标 | 值 |
|------|-----|
| 收盘价 | {{dji_close}} |
| 涨跌幅 | {{dji_change}}% |
| 分析结论 | {{dji_analysis}} |

### 纳斯达克综合指数 (IXIC)
| 指标 | 值 |
|------|-----|
| 收盘价 | {{nasdaq_close}} |
| 涨跌幅 | {{nasdaq_change}}% |
| 分析结论 | {{nasdaq_analysis}} |

### 标普500 (SPX)
| 指标 | 值 |
|------|-----|
| 收盘价 | {{spx_close}} |
| 涨跌幅 | {{spx_change}}% |
| 分析结论 | {{spx_analysis}} |

---

## 二、昨日A股走势

### 上证指数 (SH000001)
| 指标 | 值 |
|------|-----|
| 收盘价 | {{sh_close}} |
| 涨跌幅 | {{sh_change}}% |
| 分析结论 | {{sh_analysis}} |

### 深证成指 (SZ399001)
| 指标 | 值 |
|------|-----|
| 收盘价 | {{sz_close}} |
| 涨跌幅 | {{sz_change}}% |
| 分析结论 | {{sz_analysis}} |

---

## 三、昨日港股走势

### 恒生指数 (HSI)
| 指标 | 值 |
|------|-----|
| 收盘价 | {{hsi_close}} |
| 涨跌幅 | {{hsi_change}}% |
| 分析结论 | {{hsi_analysis}} |

---

## 四、市场新闻摘要

### 美股新闻
{{us_market_news}}

### A股新闻
{{cn_market_news}}

### 港股新闻
{{hk_market_news}}

---

## 五、资金流向

| 市场 | 主力净流入 | 北向资金 | 南向资金 |
|------|-----------|---------|---------|
| A股 | {{cn_net_flow}} | {{cn_south_north}} | - |
| 港股 | {{hk_net_flow}} | - | {{hk_south_north}} |

---

## 六、周线技术分析

> **注意**：`getMCPAnalysis` 周线结果不含 RSI/MACD。下列占位符无数据时填「暂无」，勿编造。

### 价格概况
{{weekly_price_summary}}

### 关键支撑与阻力

| 类型 | 位置（港币） | 重要程度 |
|------|------------|---------|
| 短期阻力 | {{weekly_resistance_short}} | ★★★ |
| 中期阻力 | {{weekly_resistance_mid}} | ★★★ |
| 长期阻力 | {{weekly_resistance_long}} | ★★ |
| 短期支撑 | {{weekly_support_short}} | ★★★ |
| 中期支撑 | {{weekly_support_mid}} | ★★★ |
| 长期支撑 | {{weekly_support_long}} | ★★ |

### 均线系统

| 均线 | 位置（港币） | 状态 |
|------|------------|------|
| MA5 | {{weekly_MA5}} | {{weekly_MA5_status}} |
| MA20 | {{weekly_MA20}} | {{weekly_MA20_status}} |
| MA60 | {{weekly_MA60}} | {{weekly_MA60_status}} |
| MA120 | {{weekly_MA120}} | {{weekly_MA120_status}} |

### 趋势判断

| 周期 | 判断 | 说明 |
|------|------|------|
| 短期（1-5日） | {{weekly_short_term}} | {{weekly_short_reason}} |
| 中期（1-4周） | {{weekly_mid_term}} | {{weekly_mid_reason}} |
| 长期（1-3月） | {{weekly_long_term}} | {{weekly_long_reason}} |

### 技术形态与指标

- **形态**：{{weekly_pattern}}
- **RSI（14）**：{{weekly_RSI}}
- **MACD**：{{weekly_MACD}}

### 操作建议

{{weekly_trading_suggestion}}

---

## 七、综合预判

```json
{
  "overall_sentiment": "bullish/bearish/neutral",
  "us_market": "positive/negative/mixed",
  "cn_market": "positive/negative/mixed",
  "hk_market": "positive/negative/mixed",
  "key_risks": ["风险点1", "风险点2"],
  "key_opportunities": ["机会点1", "机会点2"],
  "recommended_action": "偏向买入/观望/谨慎"
}
```

---

## 八、机器人状态确认

| 机器人类型 | 数量 | 状态 |
|-----------|-----|------|
| DCA交易机器人 | {{dca_count}} | {{dca_status}} |
| GRID网格机器人 | {{grid_count}} | {{grid_status}} |
| SmartTrade机器人 | {{smarttrade_count}} | {{smarttrade_status}} |
| HDG对冲机器人 | {{hdg_count}} | {{hdg_status}} |

---

## 九、操作建议

### 今日重点关注
{{key_watch_points}}

### 需要调整的机器人配置
{{adjustments_needed}}

### 风险提示
{{risk_warnings}}

---

**报告生成人**: geegoo-agent
**下次更新**: 次交易日 8:00
