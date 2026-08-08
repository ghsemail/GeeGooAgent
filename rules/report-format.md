# 盘前报告格式与 API 校验

## 市场盘前 (`premarket_market`)

**Skill:** `premarket_market` — 单次运行一个市场（CN/HK/US），每日每市场全局 **1 份**。

**API:** `create_market_premarket_report`

| 字段 | 类型 | 说明 |
|------|------|------|
| `mcp_token` | string | 用户令牌 |
| `market` | string | `CN` / `HK` / `US` |
| `report` | string | 报告正文，非空 |
| `summary` | string | 可选，≤200 字摘要 |
| `result` | string | 可选，`long` / `short` / `neutral` |
| `confidence` | string | 可选，`high` / `medium` / `low` |
| `report_date` | string | 可选，`YYYY-MM-DD`，默认当天 |

**模板:** `skills/premarket_market/template.md` — 正文三章（指数概览 / 市场新闻解读 / 市场综合判断），仅当前市场指数与新闻。

**正文格式（`report` 字段）:**

1. **指数概览** — 当前市场各指数 1–2 句自然段，禁止 Markdown 表格  
2. **市场新闻解读** — 3–5 条要点（禁止发布时间/🕐）；必须有 **新闻面判断**（偏多/偏空/中性 + 理由）  
3. **市场综合判断** — 综合结论段落 +「主要风险」「今日关注」列表；**禁止在正文写市场情绪、置信度**  
4. **脚注** — `*报告由 GeeGoo 智能体市场盘前 skill 生成*`（不要写 geegoo-agent 版本号或下次更新时间）

**禁止写入 `report` 正文:** `#` 标题行、生成时间、市场代码、`report_date` 等元数据（由 API 字段与 App 页眉展示）。

**结构化字段（API，非正文重复）:**

| 字段 | 用途 |
|------|------|
| `result` | `long` / `short` / `neutral` → App「市场情绪」 |
| `confidence` | `high` / `medium` / `low` → App「执行度」 |
| `summary` | ≤200 字一句话 → App「摘要」区 |

**LLM 合成:** 在 `save_local_report` / `create_market_premarket_report` 前，用指数与新闻证据 + 模板生成完整 `report`，并输出 `result` / `confidence` / `summary`。LLM 不可用或失败时回退规则草稿（结构同上）。

**本地留档:**

```text
{workspace_root}/reports/<YYYYMMDD>/market-<MARKET>-market_premarket.md
```

---

## 个股盘前 (`premarket_stock`)

**Skill:** `premarket_stock` — 引用市场报告后，为 attitude 订阅标的逐股生成报告。

### create_stock_premarket_report 必填字段

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
| `reason` | string | 判定依据（≥80字，含具体参数引用），非空 |
| `suggestion` | string | `buy` / `sell` / `hold` |
| `report` | string | 报告原文，非空 |
| `market_premarket_report_id` | string | 关联当日市场盘前报告 ID |

建议同时提供：`summary`、`support`、`resistance`。

### 报告模板（七章）

模板文件：`skills/premarket_stock/template.md`

1. **市场背景** — 引用 `get_market_premarket_report` **摘要**，不重复三市场指数、不粘贴整份市场报告
2. **个股新闻** — 3–5 条要点，禁止发布时间/🕐
3. **资金流向与分布**（**必须有定量分析结论**）
4. **周线技术分析** — 自然段描述，禁止 Markdown 表格；RSI/MACD 无数据填「暂无」
5. **Bot 盘前态度**
6. **综合研判** — 自然段；**禁止在正文写 result/confidence/suggestion**
7. **今日重点关注 / 风险提示** — 列表

**脚注：** `*报告由 GeeGoo 智能体个股盘前 skill 生成*`

**结构化字段（API，非正文重复）:**

| 字段 | 用途 |
|------|------|
| `result` | `long` / `short` / `neutral` → App「方向判断」 |
| `confidence` | `high` / `medium` / `low` → App「置信度」 |
| `suggestion` | `buy` / `sell` / `hold` → App「建议操作」 |
| `reason` | ≥80 字判定依据 → App 补充说明区 |
| `summary` | ≤200 字 → App「摘要」区 |

**LLM 合成:** 在 `save_local_report` / `create_stock_premarket_report` 前，用证据 + 模板生成完整 `report`，并输出 `result` / `confidence` / `reason` / `suggestion` / `summary`。LLM 不可用或失败时回退规则草稿（结构同上）。

### 周线技术分析（API 实际字段）

`getMCPAnalysis` `period=weekly` **不返回 RSI/MACD**。模板若含 RSI/MACD 占位符，填「暂无」，勿编造。

实际可用字段：支撑位、阻力位、均线位置、趋势判断、成交量信号、操作建议。

### 资金分布格式化

禁止写 raw JSON。推荐格式：

```text
超大单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
大单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
中单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
小单净流入：+X.X亿（滞留：+X.X亿 / 撤离：-X.X亿）
更新时间：YYYY-MM-DD HH:MM:SS
```

### 综合判断质量要求

1. **reason 必须包含具体参数**，如「指数偏正面；资金面积极：主力净流入+3.5亿」
2. **置信度依据**：4+ 维度同向 → high；3 维度 → medium；信号冲突 → low
3. **禁止空洞表述**：禁止「综合来看偏乐观」「建议观望」等无数据分析的结论

### 本地留档路径

```text
{workspace_root}/reports/<YYYYMMDD>/<code>-premarket.md
```

### 飞书推送

完整 Markdown 存本地；推送仅发摘要（约 2000 字符限制）。港股个股新闻可无数据，不阻塞流程。
