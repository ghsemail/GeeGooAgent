---
name: pre_market
description: 盘前准备 — 交易日 8:00 生成预判报告（指数+新闻+个股+写库）
version: "1.0.0"
---

# pre_market Skill Pack

GeeGoo 盘前自动化工作流。由 `WorkflowRunner` 按 `manifest.yaml` 步骤表驱动；LLM 仅负责解析与报告合成（Step 12）。

## 资产文件

| 文件 | 说明 |
|------|------|
| `manifest.yaml` | Tool 白名单、workflow 步骤、bundled 脚本 |
| `workflow.md` | 完整业务流程（阶段 A/B） |
| `template.md` | 盘前报告 Markdown 模板 |

## 关联 Rules（仓库根目录 `rules/`）

- `api-routing.md` — geegoo mcp 5700 路由
- `attitude-mapping.md` — attitude → result 映射
- `report-format.md` — create_pre_market_report 必填字段

## Bundled 脚本（`skills/bundled/`）

- `finance-news` — 市场新闻 US/CN/HK
- `eastmoney-news` — A股/个股新闻搜索
- `free-stock-global-quotes-news` — A股个股新闻备选

## 运行

```bash
geegoo run pre_market --config config.json
geegoo run pre_market --dry-run --config config.json
geegoo resume --session <id> --config config.json
geegoo chat --config config.json
```

## 非交易日

`check_trading_day` 返回 `false` 时 workflow 立即完成，不生成个股报告。

## 已知 API 状态（2026-05-20）

- `getCapitalFlow`：已恢复，`period=DAY`
- `getPreMarketReports`：ObjectId 序列化已修复
- `getBotYesterdayAttitude`：404 为正常空状态 → neutral
