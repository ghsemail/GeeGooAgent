#!/usr/bin/env python3
"""Probe news worker, sources config, and refresh logs on both data nodes."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> None:
    for target, label in [("geegoo-data", "US-HK"), ("geegoo-data-cn", "CN")]:
        print(f"\n{'='*60}\n## {label} ({target})")
        print("### news-worker process")
        print(run(target, "pgrep -af news-worker || echo 'NOT RUNNING'"))
        print("### env (news-related)")
        print(run(target, "grep -E 'NEWS_|SERPAPI|BOT_SERVICE|MONGO_URI|NEWS_SOURCES' .env 2>/dev/null || grep -E 'NEWS_|SERPAPI|BOT_SERVICE|MONGO_URI|NEWS_SOURCES' */GeeGooData/.env 2>/dev/null || true"))
        remote = "/root/apps/GeeGooData" if target == "geegoo-data" else "/home/ubuntu/apps/GeeGooData"
        print("### news sources file")
        print(run(target, f"grep NEWS_SOURCES {remote}/.env 2>/dev/null; ls -la {remote}/config/news_sources*.xml 2>/dev/null"))
        print("### GET /v1/news/sources (enabled only)")
        print(
            run(
                target,
                f"curl -s http://127.0.0.1:3300/v1/news/sources | python3 -c \"import sys,json;d=json.load(sys.stdin);"
                f"print('regions:',[(r.get('market'),[s['id'] for s in r.get('sources',[]) if s.get('enabled')]) for r in d.get('regions',[])])"
                f");print('stock:',[(r.get('market'),[s['id'] for s in r.get('sources',[]) if s.get('enabled')]) for r in d.get('stock_sources',[])])\"",
            )
        )
        print("### latest refresh log")
        print(
            run(
                target,
                "curl -s -X POST http://127.0.0.1:3300/getNewsRefreshLogs -H 'Content-Type: application/json' "
                "-d '{\"limit\":2}' | python3 -c \"import sys,json;d=json.load(sys.stdin);"
                "[print(x) for x in (d if isinstance(d,list) else [d])]\"",
            )
        )
        print("### news health")
        print(run(target, "curl -s http://127.0.0.1:3300/v1/news/health | python3 -m json.tool 2>/dev/null | head -40"))


if __name__ == "__main__":
    main()
