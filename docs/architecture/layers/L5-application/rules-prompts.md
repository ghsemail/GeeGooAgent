# L5 — Rules & Prompts

## 职责

常驻指令层：身份、硬规则、报告格式——不随 Skill 变化的部分。

## 文件布局

```
prompts/
└── identity.md              # Agent 身份与核心约束

rules/
├── api-routing.md           # 5700；资金流向/报告查询路由
├── attitude-mapping.md      # bullish→long 等
├── report-format.md         # 九章盘前模板约束
├── execution-log.md         # 日志规范
├── risk-disclaimer.md       # 免责声明
├── analysis.md              # getMCPAnalysis period/name 约束
├── bot-creation.md          # Phase 6：创建前确认
└── signal-reference.md      # Phase 5：指标信号
```

## identity.md 要点

- 股票分析专员身份
- 禁止编造行情
- 禁止硬编码股票代码
- 港股新闻可无数据

## Rules 与 Tool Schema 分工


| 层级          | 约束方式       | 示例                     |
| ----------- | ---------- | ---------------------- |
| Rules       | Prompt 软约束 | 「盘前同时调 getCapitalFlow + getCapitalDistribution」 |
| Tool Schema | 硬拒绝        | 缺 `confidence` 不发给 API |
| Supervisor  | 跑后检查       | 每股都有 report_id         |


## MVP

加载全部 rules（除 bot-creation、signal-reference 可 stub）。