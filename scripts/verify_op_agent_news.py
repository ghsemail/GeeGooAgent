#!/usr/bin/env python3
"""Verify news API via deployed op_agent BFF."""
from __future__ import annotations

import json
import urllib.request

BASE = "http://146.56.225.252:8088"


def get(path: str) -> tuple[int, str]:
    req = urllib.request.Request(f"{BASE}{path}")
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            return r.status, r.read(300).decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read(300).decode("utf-8", "replace")


def main() -> None:
    for path in (
        "/op_agent/v1/data/overview?force=true",
        "/op_agent/v1/data/nodes/ashare-cn/news/sources",
        "/op_agent/v1/data/nodes/ashare-cn/news/health",
    ):
        code, body = get(path)
        print(f"{path} -> {code}")
        print(body[:200])
        print()


if __name__ == "__main__":
    main()
