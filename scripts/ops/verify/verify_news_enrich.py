#!/usr/bin/env python3
"""Verify news delete, language, attitude endpoints."""
from __future__ import annotations

import json
import urllib.request

CATALOG = "http://146.56.225.252:3210"
DATA_HK = "http://47.80.14.120:3300"
BOT = "http://118.195.135.97:3100"


def post(url: str, payload: dict, headers: dict | None = None) -> tuple[int, object]:
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json", **(headers or {})},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as resp:
        body = resp.read().decode()
        try:
            return resp.status, json.loads(body)
        except json.JSONDecodeError:
            return resp.status, body


def main() -> int:
    print("=== getStockNews attitude/language (HK node) ===")
    _, news = post(
        DATA_HK + "/getStockNews",
        {"stock_list": [{"code": "TSLA.US", "name": {"init": "Tesla"}}], "language": "en"},
    )
    if isinstance(news, list) and news:
        n = news[0]
        title = n.get("title", {})
        print("attitude:", n.get("attitude"))
        print("title.en:", title.get("en", "")[:80])
        print("title.cn:", title.get("cn", ""))
    else:
        print("no news", news)

    print("\n=== deleteNewsRefreshLogs (data direct, dry - invalid id) ===")
    _, del_res = post(DATA_HK + "/deleteNewsRefreshLogs", {"run_id": "__nonexistent__"})
    print(del_res)

    print("\n=== catalog deleteNewsRefreshLogs proxy ===")
    try:
        _, cat = post(CATALOG + "/deleteNewsRefreshLogs", {"run_id": "__nonexistent__"})
        print(cat)
    except Exception as e:
        print("catalog error (may need restart):", e)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
