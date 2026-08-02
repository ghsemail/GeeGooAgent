#!/usr/bin/env python3
"""Verify news endpoints after proxy/auth fixes."""
from __future__ import annotations

import json
import urllib.request

BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
CAT_KEY = "850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9"


def post(url: str, body: dict, bearer: str | None = None) -> tuple[int, object]:
    data = json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}
    if bearer:
        headers["Authorization"] = f"Bearer {bearer}"
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    with urllib.request.urlopen(req, timeout=60) as r:
        raw = r.read()
        try:
            return r.status, json.loads(raw)
        except json.JSONDecodeError:
            return r.status, raw[:200]


def main() -> None:
    stocks = [
        {"code": "TSLA.US", "name": {"init": "Tesla"}},
        {"code": "000858.SZ", "name": {"init": "WLY"}},
        {"code": "00700.HK", "name": {"init": "Tencent"}},
    ]
    code, data = post(
        "http://118.195.135.97:3100/getStockNews",
        {"stock_list": stocks},
        BOT_KEY,
    )
    print("bot getStockNews", code, "items", len(data) if isinstance(data, list) else data)

    code, data = post(
        "http://146.56.225.252:8088/op_catalog/getNewsRefreshLogs",
        {"limit": 2},
        CAT_KEY,
    )
    if isinstance(data, list):
        print("op_catalog logs", code, "runs", len(data), "latest", data[0].get("run_date"), data[0].get("total_news"))
    else:
        print("op_catalog logs", code, data)


if __name__ == "__main__":
    main()
