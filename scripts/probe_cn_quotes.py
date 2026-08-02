#!/usr/bin/env python3
"""Probe A-share quote/kline paths for ghsemail user."""
from __future__ import annotations

import json
import urllib.request

USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
SIG_KEY = "a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402"


def post(url: str, body: dict, key: str | None = None) -> tuple[int, object]:
    data = json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            raw = r.read()
            try:
                return r.status, json.loads(raw)
            except json.JSONDecodeError:
                return r.status, raw[:300]
    except urllib.error.HTTPError as e:
        raw = e.read()
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, raw[:300]


def main() -> None:
    cn_codes = ["000858.SZ", "002466.SZ", "600519.SH"]
    for code in cn_codes:
        c, d = post(
            "http://118.195.135.97:3100/getCurrentPrice",
            {"code": code},
            BOT_KEY,
        )
        price = d.get("price") if isinstance(d, dict) else d
        print(f"getCurrentPrice {code}: http={c} price={price}")

    for code in cn_codes[:2]:
        c, d = post(
            "http://146.56.225.252:3200/getDashboardKline",
            {"code": code, "language": "cn"},
            SIG_KEY,
        )
        n = len(d) if isinstance(d, list) else d
        print(f"getDashboardKline {code}: http={c} type={type(d).__name__} n={n}")
        if isinstance(d, list) and d:
            print("  sample:", json.dumps(d[0], ensure_ascii=False)[:200])

    c, d = post(
        "http://118.195.135.97:3100/getUserStockTrend",
        {
            "user_id": USER,
            "type": "flag",
            "frequency": "5m",
            "signal_index_list": [],
        },
        BOT_KEY,
    )
    if isinstance(d, list):
        cn = [x for x in d if str(x.get("code", "")).endswith((".SZ", ".SH"))]
        print(f"getUserStockTrend: http={c} total={len(d)} cn={len(cn)}")
        for item in cn[:3]:
            print(
                "  CN",
                item.get("code"),
                "price=",
                item.get("price"),
                "signal=",
                (item.get("signal") or item.get("flag")),
            )
    else:
        print("getUserStockTrend:", c, d)

    for code in cn_codes[:1]:
        for base in [
            "http://82.157.97.76:3300",
            "http://47.80.14.120:3300",
        ]:
            c, d = post(f"{base}/v1/market/quote", {"code": code}, None)
            print(f"data quote {base} {code}: http={c}", str(d)[:120])


if __name__ == "__main__":
    main()
