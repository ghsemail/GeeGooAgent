#!/usr/bin/env python3
"""Rewrite GeeGooAgent docs from Trading 5xxx ports to GeeGoo Go 3xxx."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ROOTS = [ROOT / "docs", ROOT / "rules", ROOT / "README.md", ROOT / "PROGRESS.md"]

SKIP_DIRS = {".git", "vendor", "node_modules", "tests", ".gomodcache-review"}
EXTS = {".md"}

REPLACEMENTS: list[tuple[str, str]] = [
    ("geegoo mcp:5700", "GeeGooBot mcp-api:3120"),
    ("geegoo mcp :5700", "GeeGooBot mcp-api :3120"),
    ("geegoo mcp:3120", "GeeGooBot mcp-api:3120"),
    ("http://0.0.0.0:5700", "http://127.0.0.1:3120"),
    ("http://localhost:5700", "http://localhost:3120"),
    ("http://<host>:5700", "http://<host>:3120"),
    (":5700", ":3120"),
    ("SignalServer** `:5800`", "GeeGooSignal** `:3210`"),
    ("SignalServer** `:3210`", "GeeGooSignal catalog-api** `:3210`"),
    ("SignalServer** `:3200`", "GeeGooSignal signal-api** `:3200`"),
    ("SignalServer` `:5800`", "GeeGooSignal catalog-api` `:3210`"),
    ("SignalServer` `:3210`", "GeeGooSignal catalog-api` `:3210`"),
    ("SignalServer` `:3200`", "GeeGooSignal signal-api` `:3200`"),
    ("SignalServer**", "GeeGooSignal**"),
    ("SignalServer`", "GeeGooSignal`"),
    ("SignalServer", "GeeGooSignal"),
    (":5800", ":3210"),
    (":5600", ":3230"),
    (":5500", ":3140"),
    ("TradingBot `mcpAPIServer`", "GeeGooBot `mcp-api`"),
    ("TradingBot mcpAPIServer", "GeeGooBot mcp-api"),
    ("TradingBot", "GeeGooBot"),
    ("TradingSignal", "GeeGooSignal"),
    ("TradingData", "GeeGooData"),
    ("geegoo mcp", "GeeGooBot mcp-api"),
    ("MarketClient (:3120)", "GeeGooBot mcp-api (:3120)"),
    ("GeeGooBotClient (:3120)", "GeeGooBot mcp-api (:3120)"),
    ("SignalClient (:3210)", "GeeGooSignal catalog-api (:3210)"),
    ("三 HTTP Client（5700/5800）", "GeeGoo 3xxx HTTP 客户端"),
    ("5700/5800", "3120/3210"),
    ("5xxx Python 老栈保留**：TradingBot `:3120`、GeeGooSignal `:3210` 等同机并行，供 App/运维与 Strangler 内部调用；**GeeGooAgent 不直连**。\n\n", ""),
    ("（5700 路径，服务端转发）", "（GeeGooBot mcp-api 内部转发）"),
    ("# geegoo mcp API 路由（5700）", "# GeeGooBot mcp-api 路由（3120）"),
    ("原 geegoo mcp（5700）", "GeeGooBot mcp-api（3120）"),
    ("**默认端口**：5700", "**默认端口**：3120"),
    ("**Base URL**：`http://<host>:5700`", "**Base URL**：`http://<host>:3120`"),
    ("单一事实来源（SSOT）**：TradingBot", "单一事实来源（SSOT）**：GeeGooBot"),
    ("5700        |", "3120        |"),
    ("pytest-httpx mock 5700", "pytest-httpx mock 3120"),
    ("http://test:5700", "http://test:3120"),
    ("| 5700", "| 3120"),
    ("**5700**", "**3120**"),
    ("@ **5700**", "@ **3120**"),
    ("（5700）", "（3120）"),
    ("# 5700", "# 3120"),
    (" 5700 ", " 3120 "),
    ("**5800**", "**3210**"),
    ("| 5800", "| 3210"),
    ("5700 或 3210", "3120 或 3210"),
    ("5700 或 5800", "3120 或 3210"),
    ("mcpAPIServer.py", "mcp-api"),
    ("GeeGooBot mcp-api（5700）", "GeeGooBot mcp-api（3120）"),
    ("GeeGooBot mcp-api 5700", "GeeGooBot mcp-api 3120"),
    ("> **5xxx Python 老栈保留**：GeeGooBot `:3120`、GeeGooSignal `:3210` 等同机并行，供 App/运维与 Strangler 内部调用；**GeeGooAgent 不直连**。\n\n", ""),
]

LOOPBACK_FIXES = [
    ("loopBackStrategy` | GeeGooBot mcp-api | GeeGooSignal** `:3210`", "loopBackStrategy` | GeeGooSignal signal-api | GeeGooSignal signal-api** `:3200`"),
    ("loopback_strategy` | GeeGooBot mcp-api | GeeGooSignal** `:3210`", "loopback_strategy` | GeeGooSignal signal-api | GeeGooSignal signal-api** `:3200`"),
    ("`loopback_strategy` | GeeGooBot mcp-api | GeeGooSignal | GeeGooBot mcp-api:3120`", "`loopback_strategy` | GeeGooSignal signal-api | — | GeeGooSignal signal-api:3200`"),
    ("`search_code` | GeeGooBot mcp-api | — | GeeGooBot mcp-api:3120`", "`search_code` | GeeGooSignal signal-api | — | GeeGooSignal signal-api:3200`"),
]


def rewrite(text: str) -> str:
    for old, new in REPLACEMENTS:
        text = text.replace(old, new)
    for old, new in LOOPBACK_FIXES:
        text = text.replace(old, new)
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text


def main() -> None:
    changed = 0
    paths: list[Path] = []
    for base in ROOTS:
        if base.is_file():
            paths.append(base)
        else:
            paths.extend(p for p in base.rglob("*") if p.is_file() and p.suffix in EXTS)
    for path in paths:
        if any(part in SKIP_DIRS for part in path.parts):
            continue
        raw = path.read_text(encoding="utf-8")
        new = rewrite(raw)
        if new != raw:
            path.write_text(new, encoding="utf-8")
            changed += 1
            print(path.relative_to(ROOT))
    print(f"updated {changed} markdown files")


if __name__ == "__main__":
    main()
