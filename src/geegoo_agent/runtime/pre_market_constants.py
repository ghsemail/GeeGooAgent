"""Pre-market workflow constants (no runtime imports)."""

from __future__ import annotations

PRE_MARKET_INDEX_ENTRIES: list[tuple[str, str]] = [
    ("道琼斯", "^DJI.US"),
    ("纳斯达克", "^IXIC.US"),
    ("上证指数", "000001.SH"),
    ("深证成指", "399001.SZ"),
    ("恒生指数", "800000.HK"),
]

PRE_MARKET_INDEX_CODES: frozenset[str] = frozenset(code for _, code in PRE_MARKET_INDEX_ENTRIES)

PRE_MARKET_NEWS_MARKETS: list[str] = ["US", "CN", "HK"]

TRADING_DAY_CHECK_CODE = "00700.HK"

DRY_RUN_SAMPLE_BOTS: list[dict[str, str]] = [
    {
        "stock_name": "腾讯控股",
        "code": "00700.HK",
        "bot_id": "dry-run-bot-1",
        "bot_name": "dry-run-bot",
        "bot_type": "DCA",
    },
]
