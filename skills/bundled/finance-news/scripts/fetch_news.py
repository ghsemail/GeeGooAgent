#!/usr/bin/env python3
"""
finance-news - 财经新闻获取脚本
数据源：
  - 新浪财经 API（lid=2516）→ A股/港股/中文市场新闻
  - CNBC RSS → 美股行业动态/宏观
  - Dow Jones RSS → 美股市场快讯

用法:
  python fetch_news.py --type US     # 美股（英文RSS，优先行业层面）
  python fetch_news.py --type CN     # A股（新浪中文）
  python fetch_news.py --type HK     # 港股（新浪中文）
  python fetch_news.py --type ALL    # 全量：美股(英) + A股(中) + 港股(中)
  python fetch_news.py --keyword AI  # 自定义关键词（新浪中文）
  python fetch_news.py --limit 10    # 条数（默认10）
"""

import argparse
import json
import re
import sys
import urllib.request
import urllib.parse
import xml.etree.ElementTree as ET
from datetime import datetime

# ===== Sina Finance API =====
SINA_API = "https://feed.mix.sina.com.cn/api/roll/get"
SINA_LID = 2516

MARKET_KEYWORDS = {
    "US": ["美股", "纳斯达克", "道琼斯", "标普", "美联储", "华尔街"],
    "CN": ["A股", "上证", "深证", "沪指", "创业板", "A股"],
    "HK": ["港股", "恒生", "H股", "港交所", "红筹"],
}

# A股后过滤关键词
CN_FILTER_KW = [
    "A股", "上证", "深证", "沪指", "创业板", "科创板",
    "沪深", "大盘", "指数", "个股",
    "沪深指数", "上证指数", "深证成指",
    "宁德时代", "比亚迪", "茅台", "五粮液", "中国平安", "招商银行",
    "中芯国际", "隆基绿能", "通威股份", "温氏股份",
]

# 东财板块聚焦页面 URL（A股行业/板块新闻）
EM_CBKJJ_URL = "https://stock.eastmoney.com/a/cbkjj.html"

# 港股后过滤关键词
HK_FILTER_KW = [
    "港股", "恒生", "H股", "港交所", "港币", "港元",
    "香港股市", "香港上市", "红筹",
    "阿里巴巴", "腾讯", "美团", "小米", "比亚迪", "京东",
    "网易", "百度", "招商银行", "中国平安", "友邦", "汇丰",
    "周大福", "药明康德", "思格新能", "新东方",
]

# ===== English RSS Sources =====
ENGLISH_SOURCES = {
    "cnbc": "https://www.cnbc.com/id/100003114/device/rss/rss.html",
    "dj": "https://feeds.a.dj.com/rss/RSSMarketsMain.xml",
}

# 关键词：过滤美股行业层面的新闻（排除纯宏观/政治）
US_INDUSTRY_KEYWORDS = [
    "AI", "tech", "stock", "market", "earnings", "chip", "semiconductor",
    "Fed", "rate", "index", "rally", "selloff", "sector", "cloud",
    "Nvidia", "Apple", "Amazon", "Meta", "Google", "Microsoft", "Tesla",
]


def fetch_sina(k="", num=10, page=1):
    params = {
        "pageid": 153, "lid": SINA_LID, "k": k,
        "num": num, "page": page, "versionNumber": "1.2.4",
    }
    url = f"{SINA_API}?{urllib.parse.urlencode(params)}"
    headers = {"User-Agent": "Mozilla/5.0", "Referer": "https://finance.sina.com.cn/"}
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            return data.get("result", {}).get("data", [])
    except Exception as e:
        print(f"[Sina] 获取失败: {e}", file=sys.stderr)
        return []


def fetch_rss(url, max_items=15):
    headers = {"User-Agent": "Mozilla/5.0"}
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            raw = resp.read()
            # 尝试 GBK/GB2312（某些中文 RSS）
            for enc in ("utf-8", "gbk", "gb2312", "latin1"):
                try:
                    text = raw.decode(enc)
                    break
                except:
                    text = raw.decode("utf-8", errors="ignore")
                    break
            root = ET.fromstring(text)
            items = list(root.iter("item"))
            return [{"title": i.findtext("title", "").strip(),
                     "desc": i.findtext("description", "").strip()[:200],
                     "source": url.split("//")[1].split("/")[0]}
                    for i in items[:max_items] if i.findtext("title")]
    except Exception as e:
        print(f"[RSS] 获取失败 {url}: {e}", file=sys.stderr)
        return []


# 宏观/政治：完全排除
SKIP = [
    "iran", "ukraine", "russia", "china", "pakistan",
    "trump", "biden", "harris", "vance",
    "white house", "congress", "senate", "house",
    "election", "campaign", "gop",
    "military", "sanction", "diplomat",
    "estate", "mortgage", "dating", "celebrit", "gossip",
    "left the u.s.", "immigrat",
    "polymarket", "maduro",
    "personal finance", "real estate", "tax cut",
]

# 强行业词：直接通过（不过滤正文）
STRONG_TITLE_KW = [
    # 公司/品牌
    "nvidia", "amazon", "apple", "meta", "google", "microsoft",
    "tesla", "openai", "amd", "berkshire", "goldman",
    # 股票/市场词
    "stock", "shares", "buy", "upgrade", "downgrade",
    "earnings", "investor", "hedge fund",
    # 技术/行业
    "chip", "semiconductor", "cloud",
    "bitcoin", "crypto", "ipo", "treasury",
    # ETF（标题有ETF通常是行业事件）
    "etf",
]

# 弱行业词：需要正文干净才能通过（WEAK_TITLE_KW 容易误匹配政治类）
WEAK_TITLE_KW = [
    # "ai","tech" → 太泛，CNBC上几乎所有新闻都能匹配
    # "fed","rate" → 政治新闻里也频繁出现（议员干预Fed）
    # "car" → right-to-repair 等政治议题会误匹配
    # 只保留真正的行业中性词
    "market", "sector", "energy", "oil", "gas",
    "breaking", "alert", "forecast",
]


def is_industry_news(title, desc):
    """过滤宏观/政治类，只保留行业/市场相关"""
    title_lower = title.lower()
    text = (title + " " + desc).lower()

    # 强行业词：直接通过
    if any(kw in title_lower for kw in STRONG_TITLE_KW):
        return True

    # 弱行业词：正文有宏观/政治词则排除
    if any(kw in title_lower for kw in WEAK_TITLE_KW):
        if any(s in text for s in SKIP):
            return False
        return True

    return False


def format_sina_news(news, label):
    lines = [f"\n## 【{label}】\n"]
    for i, item in enumerate(news, 1):
        title = item.get("title", "无标题")
        # 东财格式: time="04-25"; 新浪格式: ctime=unix timestamp
        time_val = item.get("time") or item.get("ctime", "")
        if time_val and not str(time_val).isdigit():
            time_str = time_val
        elif time_val:
            try:
                dt = datetime.fromtimestamp(int(time_val))
                time_str = dt.strftime("%m-%d %H:%M")
            except:
                time_str = time_val
        else:
            time_str = ""
        media = item.get("media_name", "")
        url = item.get("url", "")
        lines.append(f"**{i}. {title}**")
        if time_str:
            lines.append(f"   🕐 {time_str}")
        if media:
            lines.append(f"   📰 {media}")
        if url:
            lines.append(f"   🔗 {url}")
        lines.append("")
    return "\n".join(lines)


def format_en_news(news_list, label="美股行业"):
    lines = [f"\n## 【{label}】（英文）\n"]
    # news_list 已经在 fetch_us_english 里过滤过了，直接使用
    for i, item in enumerate(news_list, 1):
        lines.append(f"**{i}. {item['title']}**")
        if item["desc"]:
            lines.append(f"   {item['desc'][:200]}")
        lines.append("")
    return "\n".join(lines)


def fetch_us_english(limit=8):
    """获取美股英文行业新闻（CNBC + Dow Jones）"""
    all_news = []
    seen = set()

    for src_name, src_url in ENGLISH_SOURCES.items():
        items = fetch_rss(src_url, max_items=limit * 3)
        for item in items:
            t = item["title"]
            d = item.get("desc", "")
            if t and t not in seen and is_industry_news(t, d):
                seen.add(t)
                all_news.append(item)

    # 时间/来源自然顺序，限制条数
    return all_news[:limit]


def fetch_cn_news(keywords, num):
    """获取A股新闻：从东财板块聚焦页面提取行业/板块新闻"""
    all_news = []
    seen = set()

    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Referer": "https://stock.eastmoney.com/",
    }
    req = urllib.request.Request(EM_CBKJJ_URL, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            html = resp.read().decode("utf-8", errors="ignore")
    except Exception as e:
        print(f"[EastMoney] 页面获取失败: {e}", file=sys.stderr)
        return []

    # 从 HTML 提取文章链接及标题，按 A股关键词过滤
    pattern = r'<a[^>]+href="(https://finance\.eastmoney\.com/a/\d+\.html)"[^>]*>([^<]+)</a>'
    for url, title in re.findall(pattern, html):
        title = title.strip()
        # 跳过地缘政治类
        skip_kw = ["伊朗", "巴基斯坦", "中东", "以色列", "俄罗斯", "乌克兰", "美国", "北约", "胡塞", "黎巴嫩"]
        if any(kw in title for kw in skip_kw):
            continue
        if title and title not in seen:
            seen.add(title)
            # 从 URL 提取日期
            m = re.search(r'/(\d{4})(\d{2})(\d{2})\d+\.html$', url)
            if m:
                date_str = f"{m.group(2)}-{m.group(3)}"
            else:
                date_str = ""
            all_news.append({
                "title": title,
                "url": url,
                "time": date_str,
                "media_name": "东方财富",
            })

    all_news.sort(key=lambda x: x["url"], reverse=True)
    return all_news[:num]


def fetch_hk_news(keywords, num):
    """获取港股新闻：取新浪全量 feed，按港股关键词过滤"""
    all_news = []
    seen = set()

    # 不带 k 参数取全量 feed（k 参数在 lid=2516 下不生效）
    for item in fetch_sina(num=50):
        t = item.get("title", "")
        intro = item.get("intro", "")
        text = t + intro
        if t and t not in seen:
            # 必须有港股相关关键词
            if any(kw in text for kw in HK_FILTER_KW):
                seen.add(t)
                all_news.append(item)

    all_news.sort(key=lambda x: int(x.get("ctime", 0)), reverse=True)
    return all_news[:num]


# ===== 个股新闻获取 =====

YAHOO_RSS_TEMPLATE = "https://feeds.finance.yahoo.com/rss/2.0/headline?s={ticker}&region=US&lang=en-US"

# A股股票代码 → 东财ann_type 映射
EM_ANN_TYPES = {
    "SH": "SHA",
    "SZ": "SZA",
    "CYB": "CYB",
}

# 解析 code 格式：00700.HK → (HK, 00700)
def parse_code(code):
    """解析代码格式，返回 (market, ticker)"""
    code = code.strip()
    if code.endswith(".HK"):
        return ("HK", code[:-3].lstrip("0"))
    elif code.endswith(".US") or code.endswith(".O") or code.endswith(".N"):
        return ("US", code.rsplit(".", 1)[0].upper())
    elif code.endswith(".SH"):
        return ("CN", code[:-3])
    elif code.endswith(".SZ"):
        return ("CN", code[:-3])
    elif code.endswith(".CYB"):
        return ("CN", code[:-4])
    else:
        # 纯数字 → 尝试判断 A股
        if len(code) == 6 and code.isdigit():
            return ("CN", code)
        return (None, code)


def fetch_yahoo_news(ticker, num=5):
    """通过 Yahoo Finance RSS 获取个股英文新闻（港股/美股）"""
    url = YAHOO_RSS_TEMPLATE.format(ticker=ticker)
    headers = {"User-Agent": "Mozilla/5.0"}
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            raw = resp.read()
            for enc in ("utf-8", "gbk", "gb2312", "latin1"):
                try:
                    text = raw.decode(enc)
                    break
                except:
                    text = raw.decode("utf-8", errors="ignore")
            root = ET.fromstring(text)
            items = list(root.iter("item"))
            result = []
            for item in items[:num]:
                title = item.findtext("title", "").strip()
                desc = item.findtext("description", "").strip()[:200]
                link = item.findtext("link", "").strip()
                if title:
                    result.append({
                        "title": title,
                        "desc": desc,
                        "url": link,
                        "source": "Yahoo Finance",
                    })
            return result
    except Exception as e:
        print(f"[Yahoo RSS] 获取失败 {ticker}: {e}", file=sys.stderr)
        return []


def fetch_cn_announcement(stock_code, market="SH", num=5):
    """通过东财公告 API 获取 A股个股公告"""
    # market: SH→SHA, SZ→SZA, CYB→CYB
    ann_type = EM_ANN_TYPES.get(market, "SHA,SZA,CYB")
    url = (
        f"https://np-anotice-stock.eastmoney.com/api/security/ann"
        f"?sr=-1&page_size={num}&page_index=1&stock={stock_code}&ann_type={ann_type}"
    )
    headers = {"User-Agent": "Mozilla/5.0"}
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.loads(resp.read().decode("utf-8"))
            items = data.get("data", {}).get("list", [])
            result = []
            for item in items:
                result.append({
                    "title": item.get("title", ""),
                    "time": item.get("notice_date", "")[:10],
                    "url": f"https://data.eastmoney.com/notices/detail/{item.get('art_code', '')}",
                    "source": "东方财富公告",
                })
            return result
    except Exception as e:
        print(f"[EM Ann] 获取失败 {stock_code}: {e}", file=sys.stderr)
        return []


def fetch_stock_news(stock_name, code, num=5):
    """
    获取个股新闻，按市场调用不同数据源：
    - US: Yahoo Finance RSS (AAPL.US 格式) ✅
    - HK: Yahoo Finance (TCEHY ADR) ⚠️ 有限（Yahoo 不直接支持港股 ticker）
    - CN: 东财公告 API ⚠️ 有限（stock 参数不过滤，返回全量公告）

    返回 dict: {HK: [...], US: [...], CN: [...]}（每项可能为空列表）
    """
    market, ticker = parse_code(code)
    result = {"HK": [], "US": [], "CN": []}

    if market == "HK":
        # Yahoo Finance 对港股 ticker 不返回真实个股新闻，返回通用宏观
        # 尝试 ADR（腾讯→TCEHY、阿里→BABA 等）
        ADR_MAP = {
            "00700": "TCEHY", "09988": "BABA", "03690": "MPNGF",
            "01810": "XRTXF", "02382": "SNHNF", "01628": "KEHIY",
        }
        ticker_key = ticker.lstrip("0") or ticker
        adr = ADR_MAP.get(ticker_key, ticker_key)
        result["HK"] = fetch_yahoo_news(adr, num)
        if not result["HK"]:
            print(f"[HK News] 警告: 港股 {code} 暂无可靠免费个股新闻源，Yahoo ADR 返回内容有限", file=sys.stderr)
    elif market == "US":
        result["US"] = fetch_yahoo_news(ticker, num)
    elif market == "CN":
        # 判断是上海还是深圳
        ann_type = "SHA" if code.startswith("6") else "SZA"
        if code.startswith("4") or code.startswith("8"):
            ann_type = "CYB"
        result["CN"] = fetch_cn_announcement(ticker, ann_type, num)
        if not result["CN"]:
            print(f"[CN News] 警告: A股 {code} 暂无可靠免费个股新闻接口，东财公告 API 不按 stock 过滤", file=sys.stderr)

    return result


def main():
    parser = argparse.ArgumentParser(description="财经新闻获取（新浪 + 英文 RSS）")
    parser.add_argument("--type", choices=["US", "CN", "HK", "ALL"],
                       help="市场类型: US(美股英文) CN(中文) HK(中文) ALL(全量)")
    parser.add_argument("--keyword", type=str, help="新浪关键词搜索（逗号分隔）")
    parser.add_argument("--limit", type=int, default=8, help="条数（默认8）")
    parser.add_argument("--stock", type=str, help="个股代码（如 00700.HK、AAPL.US、600519.SH）")
    args = parser.parse_args()

    output = []

    # 个股新闻模式
    if args.stock:
        code = args.stock.strip()
        # stock_name 从 code 推断
        stock_name = code.rsplit(".", 1)[0] if "." in code else code
        news = fetch_stock_news(stock_name, code, num=args.limit)
        lines = [f"\n## 【个股新闻：{stock_name} ({code})】\n"]
        if news["HK"]:
            lines.append("\n### 港股\n")
            for i, item in enumerate(news["HK"], 1):
                lines.append(f"**{i}. {item['title']}**")
                if item.get("desc"):
                    lines.append(f"   {item['desc'][:150]}")
                if item.get("url"):
                    lines.append(f"   🔗 {item['url']}")
                lines.append("")
        if news["US"]:
            lines.append("\n### 美股\n")
            for i, item in enumerate(news["US"], 1):
                lines.append(f"**{i}. {item['title']}**")
                if item.get("desc"):
                    lines.append(f"   {item['desc'][:150]}")
                if item.get("url"):
                    lines.append(f"   🔗 {item['url']}")
                lines.append("")
        if news["CN"]:
            lines.append("\n### A股公告\n")
            for i, item in enumerate(news["CN"], 1):
                lines.append(f"**{i}. {item['title']}**")
                if item.get("time"):
                    lines.append(f"   🕐 {item['time']}")
                if item.get("url"):
                    lines.append(f"   🔗 {item['url']}")
                lines.append("")
        if not any(news.values()):
            lines.append("（暂无数据）")
        print("\n".join(lines))
        return

    # 市场新闻模式（原有逻辑）
    if args.type in ("US", "ALL"):
        if args.type == "US":
            en_news = fetch_us_english(limit=args.limit)
            output.append(format_en_news(en_news, "美股行业动态"))
        else:  # ALL
            en_news = fetch_us_english(limit=args.limit)
            output.append(format_en_news(en_news, "美股行业动态"))

    if args.type in ("CN", "ALL") or args.keyword:
        kws = None
        if args.keyword:
            kws = [k.strip() for k in args.keyword.split(",")]
        cn_news = fetch_cn_news(kws, args.limit)
        output.append(format_sina_news(cn_news, "A股市场"))

    if args.type in ("HK", "ALL"):
        hk_news = fetch_hk_news(None, args.limit)
        output.append(format_sina_news(hk_news, "港股市场"))

    if not output:
        output.append("请指定 --type 或 --keyword 或 --stock")

    print("\n".join(output))


if __name__ == "__main__":
    main()