# Tool 全量分类索引

> **SSOT**：[interface-map.md](../../reference/geegoo-mcp/interface-map.md) — **73** HTTP 路由 × geegoo Skill × GeeGoo Agent Tool。  
> 本文不再重复 HTTP 明细；改名、新增、领域划分 **以 interface-map 为准**。

## 如何使用

| 角色 | 查什么 | 文档 |
|------|--------|------|
| GeeGoo Agent 实现 | Tool 名、client、Phase/MVP | [tool-catalog.md](../layers/L2-tools/tool-catalog.md) · `src/geegoo_agent/tools/catalog.py` |
| HTTP 路径 ↔ Tool | 总表 | [interface-map.md](../../reference/geegoo-mcp/interface-map.md) |
| 参数与示例 | 专题 | [geegoo-mcp/README.md](../../reference/geegoo-mcp/README.md) |
| Workflow 订阅 | Skill 白名单 | [skills-and-tools-taxonomy.md](./skills-and-tools-taxonomy.md) · `skills/*/manifest.yaml` |

**图例**（interface-map 与 catalog 共用）：✅ bespoke 已注册 · 📋 HTTP 透传 · 🔧 本地脚本

---

## 文档领域 → Tool 域（12）

| 领域 ID | 专题 | 典型 Tool / HTTP |
|---------|------|------------------|
| common | common.md | `get_position` · `get_bot_log_by_type` |
| trading | market/trading-data.md | `search_code` · `get_ticker` · `check_trading_day` |
| reports | market/reports.md | `get_report_bot_codes` · `create_*_report` |
| analyst | analyst/agent-analyst.md | `get_mcp_analysis` · `get_single_prompt_template` |
| strategy | strategy/ | `generate_*_strategy` · `loopback_strategy` |
| dca_bot … smart_reminder | bot/*.md · reminder/*.md | `create_*_bot` · `list_*` · `get_*_log` |

完整列表见 **[interface-map.md](../../reference/geegoo-mcp/interface-map.md)**。

---

## GeeGoo Agent 本地 Tool（非 HTTP）

| Tool | 说明 | 默认 Skill |
|------|------|------------|
| `fetch_market_news` | bundled finance-news | pre_market |
| `fetch_stock_news` | bundled eastmoney-news 等 | pre_market |
| `fetch_global_quote` | bundled global-quotes | on_demand |
| `save_local_report` | 工作区 `reports/` | 三个报告 Skill |
| `send_feishu_summary` | webhook | 报告 Skill（可选） |
| `write_execution_log` | execution-log.md | 全部 Skill |
| `read_working_state` | WorkingMemory | 全部 Skill |
| `recall_yesterday_summary` | Episodic 本地 | post_market |
| `recall_past_attitude` | jsonl | post_market |
| `compare_daily_reports` | 差分 | post_market |
| `wait_for_human` | Bot 创建前确认 | bot_manager |

---

## 按 Skill 订阅（摘要）

| Skill | 主要 Tool 组 |
|-------|-------------|
| **pre_market** | `check_trading_day` · `get_report_bot_codes` · 资金/态度 · 新闻 · `get_mcp_analysis` · `create_pre_market_report` |
| **intraday** | `get_ticker` · `get_broker` · `get_position` · 盘中报告 CRUD |
| **post_market** | `get_report_bot_codes` · `get_bot_log_by_type` · `create_post_market_report` |
| **on_demand_analysis** | Common + analyst 分析原子 |
| **strategy** | 信号列表 + 策略生成 + 回测 |
| **bot_manager** | Common + bot/reminder CRUD |

明细白名单：`skills/pre_market/manifest.yaml` 等。

---

## 代码入口

```text
src/geegoo_agent/tools/
├── catalog.py      # HttpToolSpec 注册表
├── bootstrap.py    # bespoke + HTTP 注册
├── perceive.py     # check_trading_day, get_report_bot_codes, …
├── analyze.py      # 资金/态度/MCP 分析
├── act_reports.py  # 报告写库 + save_local_report
└── http_api.py     # catalog 自动生成 HTTP Tool
```

---

## 相关文档

- [skills-and-tools-taxonomy.md](./skills-and-tools-taxonomy.md)
- [geegoo-api-routing.md](./geegoo-api-routing.md)
- [../layers/L2-tools/tool-catalog.md](../layers/L2-tools/tool-catalog.md)
