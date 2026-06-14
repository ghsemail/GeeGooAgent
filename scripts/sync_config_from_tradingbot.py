"""Build geegoo-agent config.json from TradingBot API keys (no mcp_token).

Usage:
    python scripts/sync_config_from_tradingbot.py \\
        --tradingbot D:/Geegoo/TradingBot \\
        --output config.json
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "src"))

from geegoo_agent.infra.tradingbot_sync import build_config


def main() -> None:
    parser = argparse.ArgumentParser(description="Sync geegoo-agent config from TradingBot")
    parser.add_argument("--tradingbot", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--host", default="118.195.135.97", help="GeeGoo API host")
    args = parser.parse_args()
    config = build_config(args.tradingbot.resolve(), base_host=args.host)
    args.output.write_text(json.dumps(config, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"Wrote {args.output} (run geegoo setup to add mcp_token and LLM token_key)")


if __name__ == "__main__":
    main()
