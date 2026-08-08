# 盘中交易决策报告

**生成时间**: {{timestamp}}
**股票**: {{stock_name}} ({{code}})
**机器人**: {{bot_name}} / {{bot_type}} / {{bot_id}}

---

## 一、决策信息

| 字段 | 值 |
|------|-----|
| 检查频率 | {{frequency}} |
| 本轮信号 | {{trade_type}} |
| 决策结果 | {{result_cn}} |
| 置信度 | {{confidence_cn}} |

---

## 二、分析依据

### 2.1 盘前报告参考

- **盘前判断**: {{pre_result_cn}}
- **盘前置信度**: {{pre_confidence_cn}}
- **盘前依据**: {{pre_reason}}
- **支撑 / 阻力**: {{pre_support}} / {{pre_resistance}}

### 2.2 当前持仓

- **持仓摘要**: {{position_summary}}

### 2.3 资金分布

{{capital_distribution_summary}}

### 2.4 小时级分析

{{mcp_analysis_summary}}

### 2.5 最新价

- **价格来源**: {{price_source}}
- **参考价**: {{reference_price}}

---

## 三、本次决策

### 3.1 本次结果

**{{result_cn}}**（置信度：{{confidence_cn}}）

### 3.2 判定依据

{{reason}}

### 3.3 与本轮信号的关系

- **本轮信号**: {{trade_type}}
- **是否赞成执行**: {{result_cn}}

---

## 四、风险控制

{{risk_control}}
