#!/usr/bin/env python3
"""E2E verify Bot news routing after CN URL fix."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:/Users/ghsemail/.cursor/skills/remote-deploy/deploy.json")
BOT_DIR = "/home/ubuntu/apps/GeeGooBot"
CN_URL = "http://82.157.97.76:3300"


def ssh(name: str) -> paramiko.SSHClient:
    s = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))["targets"][name]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    return c


def run(c: paramiko.SSHClient, cmd: str, timeout: int = 120) -> tuple[int, str]:
    _, o, e = c.exec_command(cmd, timeout=timeout)
    return o.channel.recv_exit_status(), (o.read() + e.read()).decode("utf-8", errors="replace")


def fetch_mcp_token(bot: paramiko.SSHClient) -> str:
    # Prefer GeeGooAgent config (same token used in production E2E).
    agent = ssh("geegoo-agent")
    _, out = run(
        agent,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token',''))\"",
        timeout=30,
    )
    agent.close()
    token = out.strip().splitlines()[-1] if out.strip() else ""
    if token:
        return token

    mongo_cmd = rf"""
cd {BOT_DIR} && set -a && source .env && set +a && python3 - <<'PY'
import os, sys, subprocess
try:
    from pymongo import MongoClient
except ImportError:
    subprocess.check_call([sys.executable, '-m', 'pip', 'install', 'pymongo', '-q'])
    from pymongo import MongoClient
uri = os.environ.get('GEEGOO_BOT_MONGO_URI','')
dbn = os.environ.get('GEEGOO_BOT_MONGO_DB','QT_DB')
c = MongoClient(uri, serverSelectionTimeoutMS=8000)
doc = c[dbn]['user'].find_one({{'mcp.mcp_token': {{'$exists': True, '$ne': ''}}}}, {{'mcp.mcp_token': 1}})
print((doc or {{}}).get('mcp', {{}}).get('mcp_token',''))
PY
"""
    code, out = run(bot, mongo_cmd, timeout=120)
    token = out.strip().splitlines()[-1] if out.strip() else ""
    if not token:
        raise SystemExit(f"failed to load mcp_token from mongo (exit {code}): {out[-500:]}")
    return token


def main() -> int:
    failed = False
    bot = ssh("geegoo-bot")

    _, out = run(
        bot,
        f"grep '^GEEGOO_DATA_CN_HTTP_URL=' {BOT_DIR}/.env && "
        f"curl -s -m 10 -o /dev/null -w 'cn_health:%{{http_code}}' {CN_URL}/health && echo",
    )
    print("=== CN routing ===\n", out.strip())
    if "cn_health:200" not in out:
        failed = True

    _, bearer = run(bot, f"grep '^GEEGOO_BOT_MCP_API_KEY=' {BOT_DIR}/.env | cut -d= -f2-")
    bearer = bearer.strip()
    if not bearer:
        agent = ssh("geegoo-agent")
        _, out = run(
            agent,
            "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
            "print(c.get('geegoo_api_key',''))\"",
            timeout=30,
        )
        agent.close()
        bearer = out.strip().splitlines()[-1] if out.strip() else ""
    mcp_token = fetch_mcp_token(bot)
    print("mcp_token: ok")

    tests = [
        ("CN market news", "/getMarketNews", {"mcp_token": mcp_token, "market": "CN", "limit": 1}),
        ("600519 stock news", "/getStockNews", {"mcp_token": mcp_token, "code": "600519.SH", "limit": 1}),
        ("HK market news", "/getMarketNews", {"mcp_token": mcp_token, "market": "HK", "limit": 1}),
        ("00700 stock news", "/getStockNews", {"mcp_token": mcp_token, "code": "00700.HK", "limit": 1}),
        ("US market news", "/getMarketNews", {"mcp_token": mcp_token, "market": "US", "limit": 1}),
        ("AAPL stock news", "/getStockNews", {"mcp_token": mcp_token, "code": "AAPL.US", "limit": 1}),
    ]
    for label, path, payload in tests:
        body = json.dumps(payload, ensure_ascii=False).replace("'", "'\\''")
        cmd = (
            f"curl -s -m 90 -w '\\nHTTP %{{http_code}}\\n' "
            f"-H 'Authorization: Bearer {bearer}' -H 'Content-Type: application/json' "
            f"-d '{body}' http://127.0.0.1:3120{path}"
        )
        code, out = run(bot, cmd, timeout=120)
        print(f"\n=== {label} ===\n{out[:700]}")
        if "HTTP 200" not in out:
            failed = True

    bot.close()
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
