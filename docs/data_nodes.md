# GeeGooAgent 数据节点 BFF

Monday「数据管理」通过 Agent Runtime（BFF）访问 GeeGooData，不把 Data API 直接暴露给浏览器。

## 三节点（`config.json` → `data_nodes`）

| `id` | 用途 | `regions` |
|------|------|-----------|
| `ashare-cn` | A 股节点 | `CN` |
| `us-hk` | 港美节点 | `US`, `HK` |
| `crypto` | 虚拟货币节点 | `CRYPTO` |

示例见仓库根目录 `config.example.json`。加密节点在统一新闻源目录落地后同样承载共享源（`websearch`、`x_public`、`redfox_toutiao`、`redfox_wechat`）的列表、开关与健康探活。

## 新闻源 BFF 路由（透传 GeeGooData）

| BFF | 上游 |
|-----|------|
| `GET /v1/data/nodes/{id}/news/sources` | `GET /v1/news/sources` |
| `PATCH /v1/data/nodes/{id}/news/sources` | `PATCH /v1/news/sources` |
| `GET /v1/data/nodes/{id}/news/health` | `GET /v1/news/health` |
| `GET /v1/data/overview` | 多节点聚合探活与新闻源计数 |
| `GET /v1/data/nodes?probe=true` | 单节点探活快照 |

PATCH 会清空 overview 缓存，以便计数与探活状态及时刷新。

### 透传字段（GeeGooData `feat/stock-and-market-news-sources`）

响应在 upstream JSON 基础上增加 `node_id`。下列字段由 Data API 提供，BFF **原样转发**（具体命名以 Data PR 为准，便于对齐调整）：

- **`catalog`**：`[]{ id, label, enabled, shared?, category?, layers?, pools? }` 统一目录
- **`pools`**：宏观 7 池元数据（`cn_a`、`hk`、`us`、`commodity_fx`、`bond`、`crypto`、`volatility`）
- **`regions`**：保留给既有 CN/US/HK 采集布局；当 `catalog` 非空时 overview **按 `source_id` 去重计数**，不再把同一 id 的市场层与个股层算两次

### Overview 计数

- 单节点 `news.enabled_sources` / `healthy_sources`：对该节点启用源按 **`source_id` 去重**
- 全局 `summary.sources_total` / `sources_healthy`：跨三节点再按 **`source_id` 去重**（共享源只计一次）