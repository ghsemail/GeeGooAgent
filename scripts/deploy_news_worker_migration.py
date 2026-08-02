#!/usr/bin/env python3
"""Deploy GeeGooData news worker migration to HK/US and CN nodes."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmds: list[str]) -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=90,
    )
    for cmd in cmds:
        print(f"[{target}] $ {cmd}")
        _, o, e = c.exec_command(cmd, timeout=300)
        out = (o.read() + e.read()).decode()
        if out.strip():
            print(out)
    c.close()


def patch_env(target: str, bot_key: str) -> None:
    remote_dir = json.loads(DEPLOY.read_text(encoding="utf-8"))["targets"][target]["remote_dir"]
    cmd = (
        f"cd {remote_dir} && "
        "git fetch origin main && git reset --hard origin/main && "
        "grep -q GEEGOO_DATA_MONGO_URI .env 2>/dev/null || echo GEEGOO_DATA_MONGO_URI=mongodb://127.0.0.1:27017 >> .env && "
        "grep -q GEEGOO_BOT_SERVICE_API_URL .env 2>/dev/null || echo GEEGOO_BOT_SERVICE_API_URL=http://118.195.135.97:3140 >> .env && "
        f"grep -q '^GEEGOO_BOT_SERVICE_API_KEY=' .env 2>/dev/null || echo GEEGOO_BOT_SERVICE_API_KEY={bot_key} >> .env && "
        "grep -q GEEGOO_DATA_NEWS_REFRESH_ENABLED .env 2>/dev/null || echo GEEGOO_DATA_NEWS_REFRESH_ENABLED=true >> .env && "
        "grep -q market_capabilities.us-hk .env 2>/dev/null || echo GEEGOO_DATA_MARKET_CAPABILITIES_FILE=config/market_capabilities.us-hk.xml >> .env && "
        "grep -q news_sources.us-hk .env 2>/dev/null || echo GEEGOO_DATA_NEWS_SOURCES_FILE=config/news_sources.us-hk.xml >> .env && "
        "bash start.sh restart && sleep 3 && bash start.sh status && "
        "./bin/news-worker -once 2>&1 | tail -8"
    )
    run(target, [cmd])


def stop_python_refresh() -> None:
    run(
        "geegoo-tradingdata",
        [
            "pkill -9 -f 'python.*RefreshNews.py' || true",
            "pgrep -af RefreshNews || echo REFRESH_STOPPED",
        ],
    )


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=bot_ssh["host"],
        username=bot_ssh["user"],
        password=bot_ssh.get("password"),
        timeout=60,
    )
    _, o, _ = c.exec_command("grep ^GEEGOO_BOT_SERVICE_API_KEY= /home/ubuntu/apps/GeeGooBot/.env | cut -d= -f2-")
    bot_key = o.read().decode().strip()
    c.close()
    if not bot_key:
        raise SystemExit("missing GEEGOO_BOT_SERVICE_API_KEY on bot host")

    run(
        "geegoo-bot",
        [
            "cd /home/ubuntu/apps/GeeGooBot && git fetch origin main && git reset --hard origin/main",
            "cd /home/ubuntu/apps/GeeGooBot && printf '2\\n' | bash start.sh",
            "sleep 2; curl -sf http://127.0.0.1:3140/health && echo",
        ],
    )

    patch_env("geegoo-data", bot_key)
    patch_env("geegoo-data-cn", bot_key)
    stop_python_refresh()

    run(
        "geegoo-data",
        [
            'curl -s -X POST http://127.0.0.1:3300/getStockNews -H "Content-Type: application/json" '
            '-d \'{"stock_list":[{"code":"AAPL.US","name":{"init":"Apple"}}]}\' | head -c 300',
            "echo",
            'curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H "Content-Type: application/json" '
            '-d \'{"limit":1}\' | head -c 250',
            "echo",
        ],
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
