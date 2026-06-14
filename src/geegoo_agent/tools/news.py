"""News fetching tools via bundled scripts."""

from __future__ import annotations

import subprocess
import sys
from pathlib import Path
from typing import Literal

from pydantic import BaseModel, Field

from geegoo_agent.tools.base import BaseTool
from geegoo_agent.tools.types import ToolCategory, ToolContext, ToolResult

MarketType = Literal["US", "CN", "HK"]


def _project_root(ctx: ToolContext) -> Path:
    if ctx.project_root is not None:
        return ctx.project_root
    return Path(__file__).resolve().parents[3]


def _run_script(ctx: ToolContext, script_rel: str, args: list[str]) -> str:
    root = _project_root(ctx)
    script = root / script_rel
    if not script.exists():
        raise FileNotFoundError(f"bundled script not found: {script_rel}")
    proc = subprocess.run(
        [sys.executable, str(script), *args],
        capture_output=True,
        text=True,
        timeout=120,
        cwd=str(root),
    )
    if proc.returncode != 0:
        stderr = proc.stderr.strip() or proc.stdout.strip()
        raise RuntimeError(f"script failed ({script_rel}): {stderr[:500]}")
    return proc.stdout.strip()


class FetchMarketNewsParams(BaseModel):
    market: MarketType = Field(description="US, CN, or HK market news")
    limit: int = Field(default=8, ge=1, le=20)


class FetchStockNewsParams(BaseModel):
    code: str
    stock_name: str
    limit: int = Field(default=5, ge=1, le=20)


class FetchMarketNewsTool(BaseTool):
    name = "fetch_market_news"
    description = "Fetch market news via bundled finance-news / eastmoney scripts."
    category = ToolCategory.PERCEPTION
    input_model = FetchMarketNewsParams

    def run(self, params: FetchMarketNewsParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped market news {params.market}",
                data={"market": params.market, "items": [], "text": ""},
            )
        if params.market == "CN":
            try:
                text = _run_script(
                    ctx,
                    "skills/bundled/eastmoney-news/search.py",
                    ["今日A股市场新闻", "--limit", str(params.limit)],
                )
                source = "eastmoney"
            except Exception:
                text = _run_script(
                    ctx,
                    "skills/bundled/finance-news/scripts/fetch_news.py",
                    ["--type", "CN", "--limit", str(params.limit)],
                )
                source = "finance-news"
        else:
            text = _run_script(
                ctx,
                "skills/bundled/finance-news/scripts/fetch_news.py",
                ["--type", params.market, "--limit", str(params.limit)],
            )
            source = "finance-news"
        return ToolResult(
            status="ok",
            summary=f"Fetched {params.market} market news via {source}",
            data={"market": params.market, "text": text, "source": source},
        )


class FetchStockNewsTool(BaseTool):
    name = "fetch_stock_news"
    description = "Fetch stock-specific news (eastmoney primary, fallbacks)."
    category = ToolCategory.PERCEPTION
    input_model = FetchStockNewsParams

    def run(self, params: FetchStockNewsParams, ctx: ToolContext) -> ToolResult:
        if ctx.dry_run:
            return ToolResult(
                status="dry_run",
                summary=f"dry-run: skipped stock news {params.code}",
                data={"code": params.code, "text": "", "source": "dry-run"},
            )
        query = f"{params.stock_name}股票新闻"
        source = "eastmoney"
        try:
            text = _run_script(
                ctx,
                "skills/bundled/eastmoney-news/search.py",
                [query, "--limit", str(params.limit)],
            )
        except Exception:
            if params.code.endswith((".SZ", ".SH")):
                source = "akshare"
                text = _run_script(
                    ctx,
                    "skills/bundled/free-stock-global-quotes-news/scripts/news.py",
                    [params.code, "--limit", str(params.limit)],
                )
            else:
                source = "finance-news-hk"
                text = _run_script(
                    ctx,
                    "skills/bundled/finance-news/scripts/fetch_news.py",
                    ["--type", "HK", "--limit", str(params.limit)],
                )
        return ToolResult(
            status="ok",
            summary=f"Stock news for {params.code} via {source}",
            data={"code": params.code, "text": text, "source": source},
        )
