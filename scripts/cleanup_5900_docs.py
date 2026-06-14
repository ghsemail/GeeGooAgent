# coding=utf-8
"""Bulk replace deprecated 5900/mk- references in GeeGooAgent markdown docs only."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
TARGETS = [ROOT / "docs", ROOT / "rules", ROOT / "skills", ROOT / "PROGRESS.md", ROOT / "README.md"]
SKIP = {"cleanup_5900_docs.py"}

REPLACEMENTS = [
    ("5700/5900", "5700"),
    ("5900/5700", "5700"),
    ("118.195.135.97:5900", "118.195.135.97:5700"),
    ("http://test:5900", "http://test:5700"),
    ("http://x:5900", "http://x:5700"),
    (":5900", ":5700"),
    (" | 5900 |", " | 5700 |"),
    ("| **5900**", "| **5700**"),
    ("mock 5900", "mock 5700"),
    ("常用 5900", "常用 5700"),
    ("marketAPIServer :5900", "mcpAPIServer :5700"),
    ("marketAPIServer（5900）", "mcpAPIServer（5700）"),
    ("Market API Server（5900）", "geegoo mcp（5700）"),
    ("GeeGoo Market API（5900）", "geegoo mcp（5700）"),
    ("MarketClient (:5900)", "MarketClient (:5700)"),
    ("`mk-...`", "`sk-...`"),
    ("MK_API_KEY", "SK_API_KEY"),
    ("5700 vs 5900", "5700"),
    ("双 Skill 双端口", "统一 geegoo mcp"),
    ("三 HTTP Client（5900/5700/5800）", "HTTP Client（5700/5800）"),
    ("docs/MCP_API_Trading.md", "docs/geegoo-mcp/market/trading-data.md"),
    ("MCP_API_Trading.md", "geegoo-mcp/market/trading-data.md"),
    ("MCP_API_Market.md", "geegoo-mcp/market/reports.md"),
    ("MCP_API_Common.md", "geegoo-mcp/common.md"),
    ("5900 `", "5700 `"),
    ("用户 bot 列表", "报告待分析标的"),
]

paths: list[Path] = []
for item in TARGETS:
    if item.is_file():
        paths.append(item)
    else:
        paths.extend(p for p in item.rglob("*.md") if p.is_file())

count = 0
for path in paths:
    if path.name in SKIP:
        continue
    text = path.read_text(encoding="utf-8")
    orig = text
    for old, new in REPLACEMENTS:
        text = text.replace(old, new)
    if text != orig:
        path.write_text(text, encoding="utf-8", newline="\n")
        count += 1
        print(path.relative_to(ROOT))

print(f"updated {count} files")
