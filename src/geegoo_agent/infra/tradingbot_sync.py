"""Sync GeeGoo MCP Bearer key from a local TradingBot checkout."""

from __future__ import annotations

import re
from pathlib import Path


def _extract_api_key(py_file: Path, var_name: str = "API_KEY") -> str:
    text = py_file.read_text(encoding="utf-8")
    match = re.search(rf'{var_name}\s*=\s*["\']([^"\']+)["\']', text)
    if not match:
        raise ValueError(f"{var_name} not found in {py_file}")
    return match.group(1)


def build_config(tradingbot_root: Path, *, base_host: str = "118.195.135.97") -> dict:
    mcp_key = _extract_api_key(tradingbot_root / "mcpAPIServer.py")
    geegoo_url = f"http://{base_host}:5700"
    return {
        "base_url": geegoo_url,
        "api_key": mcp_key,
        "geegoo_url": geegoo_url,
        "geegoo_api_key": mcp_key,
        "mcp_token": "",
        "signal_base_url": "http://146.56.225.252:5800",
        "output_dir": "/var/lib/geegoo-agent/data",
        "feishu_webhook_url": None,
        "dry_run": False,
        "max_steps": 80,
        "llm": {
            "provider": "openai",
            "token_key": "",
            "model": "",
            "temperature": 0.2,
            "max_tokens": 4096,
        },
        "sandbox": {
            "allowed_hosts": [
                base_host,
                "146.56.225.252",
                "localhost",
                "127.0.0.1",
            ]
        },
    }
