# finance-news

财经新闻获取 skill，综合三大数据源：

- **CNBC + Dow Jones RSS** → 美股行业动态（英文，行业层面）
- **东方财富·板块聚焦** → A股行业/板块新闻（中文，行业层面）
- **新浪财经 lid=2516** → 港股机构研报（中文）

无需认证，直接请求。

## 脚本

`scripts/fetch_news.py`

## 使用方式

```bash
# 美股（英文，行业层面）
python scripts/fetch_news.py --type US --limit 8

# A股（中文，行业层面）
python scripts/fetch_news.py --type CN --limit 8

# 港股（中文，机构研报）
python scripts/fetch_news.py --type HK --limit 8

# 全量：美股(英) + A股(中) + 港股(中)
python scripts/fetch_news.py --type ALL --limit 8

# 自定义关键词（新浪中文）
python scripts/fetch_news.py --keyword AI,芯片 --limit 10

# 个股新闻（美股可用，港股/A股有限）
python scripts/fetch_news.py --stock 00700.HK --limit 5
python scripts/fetch_news.py --stock AAPL.US --limit 5
python scripts/fetch_news.py --stock 600519.SH --limit 5
```

## 输出格式

```
## 【美股行业动态】（英文）
1. 新闻标题
   描述（英文摘要）

## 【A股市场】
1. 新闻标题
   🕐 时间 | 📰 来源 | 🔗 链接

## 【港股市场】
1. 新闻标题
   🕐 时间 | 📰 来源 | 🔗 链接
```

## 数据源

| 源 | 类型 | 覆盖 |
|---|---|---|
| CNBC RSS | 英文 | 美股行业动态 |
| Dow Jones RSS | 英文 | 美股市场快讯 |
| 东财板块聚焦 (stock.eastmoney.com) | 中文 | A股行业/板块新闻 |
| 新浪财经 lid=2516 | 中文 | 港股机构研报/评级 |

## 个股新闻（--stock）

| 市场 | 代码示例 | 数据源 | 状态 |
|------|---------|--------|------|
| 美股 | `AAPL.US` | Yahoo Finance RSS | ✅ 可用 |
| 港股 | `00700.HK` | Yahoo Finance (TCEHY ADR) | ⚠️ 有限 |
| A股 | `600519.SH` | 东财公告 API | ⚠️ 有限（stock 参数不过滤） |

> ⚠️ 港股/A 股免费可靠个股新闻接口暂缺，Yahoo Finance ADR 数据噪音较大。如需精准个股新闻，建议接入 Wind / 同花顺 iFinD 等付费数据源。