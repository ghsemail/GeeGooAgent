#!/usr/bin/env python3
"""
duckduckgo-search - 免费 DuckDuckGo 网页搜索（GeeGooAgent bundled skill）

用法:
  python web_search.py --query "SpaceX IPO" --limit 5
  python web_search.py --query "腾讯 新闻" --json
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import urllib.parse
import urllib.request

DDG_URL = "https://html.duckduckgo.com/html/"
USER_AGENT = "GeeGooAgent/1.0 (+https://github.com/ghsemail/GeeGooAgent)"

RESULT_LINK = re.compile(
    r'<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>'
)
RESULT_SNIPPET = re.compile(
    r'<a[^>]*class="result__snippet"[^>]*>([\s\S]*?)</a>'
)
TAG_STRIP = re.compile(r"<[^>]+>")


def clean_html(text: str) -> str:
    text = TAG_STRIP.sub("", text)
    for old, new in (("&amp;", "&"), ("&quot;", '"'), ("&#39;", "'")):
        text = text.replace(old, new)
    return text.strip()


def decode_ddg_redirect(raw: str) -> str:
    raw = raw.strip()
    if "uddg=" in raw:
        parsed = urllib.parse.urlparse(raw)
        qs = urllib.parse.parse_qs(parsed.query)
        uddg = qs.get("uddg", [""])[0]
        if uddg:
            return uddg
    return raw


def duckduckgo_search(query: str, limit: int) -> list[dict[str, str]]:
    query = query.strip()
    if not query:
        raise ValueError("query required")
    if limit <= 0:
        limit = 5
    if limit > 10:
        limit = 10

    data = urllib.parse.urlencode({"q": query}).encode("utf-8")
    req = urllib.request.Request(
        DDG_URL,
        data=data,
        headers={
            "Content-Type": "application/x-www-form-urlencoded",
            "User-Agent": USER_AGENT,
        },
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=20) as resp:
        if resp.status >= 400:
            raise RuntimeError(f"duckduckgo HTTP {resp.status}")
        html = resp.read(2 * 1024 * 1024).decode("utf-8", errors="replace")

    links = RESULT_LINK.findall(html)[:limit]
    snippets = RESULT_SNIPPET.findall(html)[:limit]
    out: list[dict[str, str]] = []
    for i, (url, title_html) in enumerate(links):
        title = clean_html(title_html)
        if not title:
            continue
        snippet = ""
        if i < len(snippets):
            snippet = clean_html(snippets[i])
        out.append(
            {
                "title": title,
                "url": decode_ddg_redirect(url),
                "snippet": snippet,
            }
        )
    return out


def format_text(query: str, results: list[dict[str, str]]) -> str:
    if not results:
        return f"web_search: no results for {query!r}"
    lines = [f"web_search: {len(results)} hit(s) for {query!r}"]
    for i, hit in enumerate(results, 1):
        lines.append(f"{i}. {hit['title']}")
        if hit.get("url"):
            lines.append(f"   {hit['url']}")
        if hit.get("snippet"):
            lines.append(f"   {hit['snippet']}")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description="DuckDuckGo web search (bundled skill)")
    parser.add_argument("--query", "-q", required=True, help="search query")
    parser.add_argument("--limit", "-n", type=int, default=5, help="max results (1-10)")
    parser.add_argument("--json", action="store_true", help="emit JSON for agent runtime")
    args = parser.parse_args()

    try:
        results = duckduckgo_search(args.query, args.limit)
    except Exception as exc:  # noqa: BLE001 — CLI entrypoint
        print(str(exc), file=sys.stderr)
        return 1

    if args.json:
        payload = {"query": args.query.strip(), "count": len(results), "results": results}
        print(json.dumps(payload, ensure_ascii=False))
    else:
        print(format_text(args.query, results))
    return 0


if __name__ == "__main__":
    sys.exit(main())
