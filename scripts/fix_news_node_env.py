#!/usr/bin/env python3
"""Ensure GeeGooData nodes have correct market/news XML env and run one refresh."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

NODES = {
    "geegoo-data": {
        "remote_dir": "/root/apps/GeeGooData",
        "capabilities": "config/market_capabilities.us-hk.xml",
        "sources": "config/news_sources.us-hk.xml",
    },
    "geegoo-data-cn": {
        "remote_dir": "/home/ubuntu/apps/GeeGooData",
        "capabilities": "config/market_capabilities.cn.xml",
        "sources": "config/news_sources.cn.xml",
    },
}


def run(target: str, cmd: str, timeout: int = 300) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def upsert_env(remote_dir: str, key: str, value: str) -> str:
    return (
        f"cd {remote_dir} && "
        f"if grep -q '^{key}=' .env 2>/dev/null; then "
        f"sed -i 's|^{key}=.*|{key}={value}|' .env; "
        f"else echo {key}={value} >> .env; fi"
    )


def main() -> int:
    for target, meta in NODES.items():
        rd = meta["remote_dir"]
        print(f"\n=== {target} ===")
        cmds = [
            upsert_env(rd, "GEEGOO_DATA_MARKET_CAPABILITIES_FILE", meta["capabilities"]),
            upsert_env(rd, "GEEGOO_DATA_NEWS_SOURCES_FILE", meta["sources"]),
            f"grep -E 'MARKET_CAPABILITIES|NEWS_SOURCES|NEWS_REFRESH' {rd}/.env",
            f"cd {rd} && bash start.sh restart && sleep 2 && pgrep -af news-worker || true",
            f"cd {rd} && set -a && source .env && set +a && ./bin/news-worker -once 2>&1 | tail -6",
            (
                "curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs "
                "-H 'Content-Type: application/json' -d '{\"limit\":1}' | "
                "python3 -c \"import sys,json;d=json.load(sys.stdin);x=d[0];"
                "print(x.get('run_date'), x.get('status'), x.get('total_news'), x.get('success_stocks'), '/', x.get('total_stocks'))\""
            ),
        ]
        for cmd in cmds:
            print("$", cmd[:100])
            print(run(target, cmd))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
