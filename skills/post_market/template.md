# 盘后分析报告

**生成时间**: {{timestamp}}
**股票**: {{stock_name}} ({{code}})
**交易日**: {{session_date}}
**机器人**: {{bot_name}} / {{bot_type}} / {{bot_id}}

---

## 一、今日行情

### 1.1 涨跌幅与盘面倾向

- **change_pct**: {{change_pct}}%
- **session_bias**: {{session_bias}}

### 1.2 小时级价格分析

{{hourly_price_excerpt}}

### 1.3 小时级信号分析

{{hourly_signal_excerpt}}

### 1.4 小时级 K 线分析

{{hourly_kline_excerpt}}

---

## 二、交易复盘

{{trade_summary}}

---

## 三、与盘前对照

| 项目 | 值 |
|------|-----|
| 盘前 report_id | {{pre_market_report_id}} |
| 盘前 result | {{pre_result}} |
| 今日 session_bias | {{session_bias}} |
| 对照 vs_pre_market | {{vs_pre_market}} |

---

## 四、经验与教训

{{experience_summary}}

---

## 五、一句话总览

{{summary}}
