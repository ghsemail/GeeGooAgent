---
name: duckduckgo-search
description: 免费 DuckDuckGo 网页搜索。search_code 无结果、查外部新闻/时事、补充个股或市场资讯时使用。
---

# duckduckgo-search

免费 DuckDuckGo HTML 搜索，无需 API Key。

## When to use

- `search_code` 在 GeeGoo 股票库无匹配
- 需要外部新闻、时事、公司背景（非结构化财经 RSS）
- `fetch_market_news` / `fetch_stock_news` 在 GeeGooData 无数据时的兜底

## 脚本

`scripts/web_search.py`

## 使用方式

```bash
# 通用搜索（默认 5 条）
python scripts/web_search.py --query "SpaceX IPO 2024"

# 指定条数（最多 10）
python scripts/web_search.py --query "腾讯 最新新闻" --limit 8

# JSON 输出（Agent runtime 调用）
python scripts/web_search.py --query "NASDAQ today" --limit 5 --json
```

## 输出格式

**人类可读：**

```
web_search: 3 hit(s)
1. Title here
   https://example.com
   Snippet text...
```

**JSON（`--json`）：**

```json
{
  "query": "SpaceX IPO 2024",
  "count": 3,
  "results": [
    {"title": "...", "url": "...", "snippet": "..."}
  ]
}
```

## Agent 集成

- Tool：`web_search`（`market` toolset）
- 运行时优先调用本 skill 脚本；Python 不可用时回退 Go 内置实现
- 配置：`config.json` → `"search": { "provider": "duckduckgo", "max_results": 5 }`

## 数据源

| 源 | 接口 | 说明 |
|---|---|---|
| DuckDuckGo | `https://html.duckduckgo.com/html/` | 免费 HTML 抓取，无 key |
