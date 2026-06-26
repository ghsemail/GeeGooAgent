# Pre-Market Workflow

**触发时间**（GeeGooAgent systemd 默认仅 A/港股盘前 **08:00**；美股请用 Hermes `盘前准备-美股` job **21:00**，见 geegoo skill `cron/market-schedules.md`）  
**目的**: 生成盘前预判报告，为当日交易决策提供依据  
**输出**: 每只股票独立 `.md` 文件（本地留档 + 数据库），按日期归档

> 工具白名单与步骤 ID 见同目录 `manifest.yaml`。API 路由见 `rules/api-routing.md`。

---

## 前置条件：校验今日是否为交易日

在执行任何步骤之前，**必须**先完成以下两步：

### 第一步：校验交易日

Tool：`check_trading_day`

```
POST /checkTradingDay  (5700)
Header: Authorization: Bearer <mk-api_key>
Body: {"mcp_token": "<mcp_token>", "code": "00700.HK"}
```

**判定规则**：

- `is_trading_day: true` → 继续阶段 A + 阶段 B
- `is_trading_day: false` → **终止 workflow**，写执行日志，跳过所有后续步骤

推荐：`00700.HK`（港股）、`AAPL.US`（美股）、`600519.SH`（A股）

### 第二步：获取股票列表

Tool：`get_report_bot_codes`

```
POST /getReportBotCodes  (5700)
Body: {"mcp_token": "<mcp_token>"}
```

| 字段 | 说明 |
|------|------|
| `code` | 股票代码 |
| `stock_name` | 股票名称 |
| `bot_id` | 机器人文档 `_id` |
| `bot_name` | 机器人名称 |
| `bot_type` | 如 `DCA` |

按 `code` 去重，`bot_id` 取首次命中。空列表 → 跳过阶段 B 并记录日志。

---

## 阶段 A：公共数据收集（全局，一次性）

### Step 1：市场指数分析

Tool：`get_mcp_analysis`（×5，`period=hourly`）

`prompt_id` 固定：`69ec7035b9ccd3d9befc6c23`

| 指数 | 代码 | name |
|------|------|------|
| 道琼斯 | ^DJI.US | 道琼斯 |
| 纳斯达克 | ^IXIC.US | 纳斯达克 |
| 上证指数 | 000001.SH | 上证指数 |
| 深证成指 | 399001.SZ | 深证成指 |
| 恒生指数 | 800000.HK | 恒生指数 |

汇总到报告「昨日美股/A股/港股走势」章节。

### Step 2：获取市场新闻

Tool：`fetch_market_news`（内置脚本降级）

**美股：**
```bash
python skills/bundled/finance-news/scripts/fetch_news.py --type US --limit 8
```

**A股（优先 eastmoney）：**
```bash
python skills/bundled/eastmoney-news/search.py "今日A股市场新闻" --limit 8
```

**A股备选：**
```bash
python skills/bundled/finance-news/scripts/fetch_news.py --type CN --limit 8
```

**港股：**
```bash
python skills/bundled/finance-news/scripts/fetch_news.py --type HK --limit 8
```

---

## 阶段 B：个股数据收集（每只股票循环）

对 `get_report_bot_codes` 每项依次执行 Step 3–8。**禁止硬编码股票代码**。

### Step 3：个股新闻

Tool：`fetch_stock_news`

```bash
python skills/bundled/eastmoney-news/search.py "<股票名称>股票新闻" --limit 5
# 备选 A股：
python skills/bundled/free-stock-global-quotes-news/scripts/news.py <code> --limit 5
# 备选港股：
python skills/bundled/finance-news/scripts/fetch_news.py --type HK --limit 5
```

港股个股新闻可能无可靠免费源，标注「暂无数据」即可。

### Step 4：资金流向

> **A 股跳过**：代码以 `.SH` / `.SZ` 结尾时，不调用本 Tool，报告写「A股资金流向暂不可用」，日志 `skipped`。

Tool：`get_capital_flow`（`period=DAY`）

取 `data` 数组中 `capital_flow_item_time` 最新一条。

### Step 5：资金分布

> **A 股跳过**：代码以 `.SH` / `.SZ` 结尾时，不调用本 Tool，报告写「A股资金分布暂不可用」，日志 `skipped`。

Tool：`get_capital_distribution`

格式化见 `rules/report-format.md`。

### Step 6：周线支撑/阻力位

Tool：`get_mcp_analysis`（`period=weekly`）

从 `analysis_result` 提取支撑/阻力，写入「技术面分析 — 周线关键价位」。

### Step 7：昨日 Bot 态度

Tool：`get_bot_yesterday_attitude`（`bot_id` 来自 Step 2）

映射见 `rules/attitude-mapping.md`。404 → `neutral`。

### Step 8：综合预判 + 写库 + 本地留档

1. 按 `template.md` 汇总（LLM 任务：解析周线 + 合成报告，Step 12）
2. Tool：`create_pre_market_report`（必填字段见 `rules/report-format.md`）
3. Tool：`save_local_report` → `{workspace_root}/reports/<YYYYMMDD>/<code>-premarket.md`
4. 可选：`send_feishu_summary`（仅摘要）

**幂等**：Step 8 前可调用 `list_today_reports` 检查同日同 code 是否已有报告。

---

## 执行日志

Tool：`write_execution_log`（每步完成后立即写入）

路径：`{workspace_root}/reports/<YYYYMMDD>/execution-log.md`

格式示例：
```
[HH:MM:SS] check_trading_day → 成功(is_trading_day=true)
[HH:MM:SS] get_report_bot_codes → 成功(N只股票)
...
[HH:MM:SS] 完成
```

时间戳用实际执行时间，禁止占位符。

---

## 流程图

```
阶段 A（一次）
├── Step 1: 5 指数 hourly 分析
└── Step 2: 市场新闻 US/CN/HK

阶段 B（每股循环）
└── for stock in get_report_bot_codes():
    ├── Step 3: 个股新闻
    ├── Step 4: get_capital_flow (DAY；A股跳过)
    ├── Step 5: get_capital_distribution（A股跳过）
    ├── Step 6: weekly get_mcp_analysis
    ├── Step 7: get_bot_yesterday_attitude
    └── Step 8: 合成报告 + create_pre_market_report + save_local_report
```
