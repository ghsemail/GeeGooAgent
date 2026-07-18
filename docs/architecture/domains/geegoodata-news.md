# 新闻聚合 — GeeGooData 统一出口（设计）

> **状态**：已采纳，**待实现**（GeeGooData → GeeGooBot mcp-api → GeeGooAgent Tool）。  
> **目标**：新闻与行情/资金同一治理——**多源在 Data 聚合**，Agent **不再本机爬外网**。  
> GeeGooData 实现细则 → [GeeGooData/docs/NEWS.md](../../../../GeeGooData/docs/NEWS.md)

## 为什么要收进 GeeGooData

| 现状 | 问题 |
|------|------|
| Agent `newsrunner` + Python `finance-news` | Python/Go 双实现，A 股行业源不一致 |
| 每台 Agent 直连 RSS/东财/新浪 | 出站、限流、doctor 难统一 |
| `eastmoney-news` bundled 未注册 Tool | 能力在 manifest 里但调不到 |

与 **现价/资金** 对齐：

```text
行情/资金：Agent → Bot :3120 → GeeGooData :3300（market_capabilities 分 CN/HK/US 节点）
新闻（目标）：Agent → Bot :3120 → GeeGooData :3300（news_sources 分市场+分源开关）
```

## 分层职责

| 层 | 职责 |
|----|------|
| **GeeGooData** | 多源拉取、去重、缓存、结构化 JSON；`news_sources.xml` 声明本节点启用哪些源 |
| **GeeGooBot mcp-api** | MCP 契约：`/getMarketNews`、`/getStockNews`；按 `market`/`code` 路由到 CN 或 US-HK Data 节点（与资金路由同 env） |
| **GeeGooAgent** | Tool 名不变：`fetch_market_news`、`fetch_stock_news`；实现改为 HTTP 转发；`web_search` 保留为 Agent 本地最后兜底（可选后续收进 Data） |

## MCP API（Bot :3120，待注册）

### `POST /getMarketNews`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `mcp_token` | string | 是 | 用户身份 |
| `market` | string | 是 | `US` \| `CN` \| `HK` \| `ALL` |
| `limit` | int | 否 | 默认 8 |
| `language` | string | 否 | `cn` / `en`（影响摘要语言，可选） |

响应 `data`：

```json
{
  "market": "CN",
  "items": [
    {
      "title": "…",
      "url": "https://…",
      "snippet": "…",
      "published_at": "2026-07-18T08:00:00Z",
      "source_id": "eastmoney_sector",
      "source_label": "东方财富·板块聚焦"
    }
  ],
  "text": "（兼容旧 workflow 的 Markdown 拼接，由 Data 生成）",
  "sources_used": ["eastmoney_sector", "sina_roll"],
  "cache_hit": false
}
```

### `POST /getStockNews`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `mcp_token` | string | 是 | |
| `code` | string | 是 | `00700.HK`、`600519.SH`、`AAPL.US` |
| `limit` | int | 否 | 默认 8 |

响应结构同上，增加 `code` 字段；`items[].source_id` 如 `eastmoney_ann`、`yahoo_rss`、`sina_roll`。

### Data 原生路由（Bot 转发目标）

| Data 路径 | 说明 |
|-----------|------|
| `POST /v1/news/market` | 市场/行业新闻 |
| `POST /v1/news/stock` | 个股新闻 |
| `GET /v1/news/sources` | 本节点 `news_sources.xml` 解析结果 + 各源 enabled |
| `GET /v1/news/health` | 各 enabled 源探活（可选） |

## 新闻源配置（类比 `market_capabilities.xml`）

文件：`GEEGOO_DATA_NEWS_SOURCES_FILE`（默认 `config/news_sources.xml`）。

按 **市场区域** 声明启用哪些 **source_id**（与 A 股/美股分节点同一思路）：

| 节点示例 | `market` 区域 | 建议启用的 `source_id` |
|----------|---------------|------------------------|
| CN `82.157.97.76` | `CN` | `eastmoney_sector`、`sina_roll`、`eastmoney_ann`（个股） |
| US-HK `47.80.14.120` | `US` | `cnbc_rss`、`dowjones_rss` |
| US-HK | `HK` | `sina_hk`、`yahoo_hk` |
| US-HK | `US`（个股） | `yahoo_rss` |

完整 source 表与 XML 示例见 GeeGooData `config/news_sources.*.xml`、`docs/NEWS.md`。

Bot 路由（与资金一致）：

| 请求 | 路由到 |
|------|--------|
| `market=CN` 或 A 股 `code` | `GEEGOO_DATA_CN_HTTP_URL` |
| `market=US/HK` 或 港美股 `code` | `GEEGOO_DATA_HTTP_URL` |

## Agent Tool 迁移

| 阶段 | `fetch_market_news` / `fetch_stock_news` |
|------|------------------------------------------|
| **当前** | `newsrunner` 本地 Go + Python fallback + `web_search` |
| **P1** | Bot MCP 优先；失败 fallback 本地（与 `generate_*` → analyze 类似） |
| **P2** | 仅 HTTP；删除 `internal/tools/newsrunner` 大部分逻辑 |
| **P3** | `web_search` 仅用于非财经泛搜索 |

空结果语义（已实现于本地，迁移后由 Data 保证）：

- API/鉴权失败 → Tool **error**
- 所有 **enabled** 源均无标题 → **error** 或 **skip**（与 `fetch_stock_news` 一致，不在 Agent 假成功）

## 与 Skill / bundled 的关系

| 资产 | 迁移后 |
|------|--------|
| `skills/bundled/finance-news` | 逻辑迁入 GeeGooData；Agent 不再 `exec` Python |
| `skills/bundled/eastmoney-news` | 作为 Data 源 `eastmoney_search`（需 API key 环境变量） |
| `pre_market` manifest `bundled:` | 可保留文档引用，或改为「依赖 Data 新闻服务」 |

## 运维与 doctor

| 检查项 | 说明 |
|--------|------|
| `GET GeeGooData /v1/news/sources` | 节点启用了哪些源 |
| `POST /v1/news/market` 探针 | US/CN/HK 各 1 条 |
| Bot `POST /getStockNews` | 00700.HK 探针 |
| Agent `geegoo doctor` | 在 Data 就绪后增加 tool 探针（替代仅测本地 newsrunner） |

## 实现顺序（建议）

1. **GeeGooData**：`news_sources.xml` + `/v1/news/*` + 移植 `finance-news` 聚合逻辑  
2. **GeeGooBot**：`getMarketNews` / `getStockNews` + CN/HK/US 路由  
3. **GeeGooAgent**：`bespoke` 改 MCP 客户端调用；更新 [tools-status.md](../layers/L2-tools/tools-status.md)  
4. **interface-map**：新增 2 条 MCP 路由文档  

## 相关文档

- [geegoo-api-routing.md](./geegoo-api-routing.md) — 3120 路由  
- [layers/L2-tools/tool-server-mapping.md](../layers/L2-tools/tool-server-mapping.md) — 部署 IP  
- GeeGooData：[NEWS.md](../../../../GeeGooData/docs/NEWS.md)、[CONFIGURATION.md](../../../../GeeGooData/docs/CONFIGURATION.md)
