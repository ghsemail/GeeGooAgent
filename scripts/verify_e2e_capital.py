#!/usr/bin/env python3
"""E2E verify capital routing (post firewall open)."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
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
        raise SystemExit(f"failed to load mcp_token from mongo (exit {code})")
    return token


def main() -> int:
    failed = False
    bot = ssh("geegoo-bot")

    # ensure direct CN URL (not tunnel)
    run(
        bot,
        f"sed -i 's|^GEEGOO_DATA_CN_HTTP_URL=.*|GEEGOO_DATA_CN_HTTP_URL={CN_URL}|' {BOT_DIR}/.env && "
        f"pkill -f 'ssh.*13300:127.0.0.1:3300' || true",
    )

    code, out = run(
        bot,
        f"curl -s -m 10 -o /dev/null -w '%{{http_code}}' {CN_URL}/health && echo",
    )
    print("=== Bot -> CN Data health ===", out.strip())
    if out.strip() != "200":
        failed = True

    _, bearer = run(bot, f"grep '^GEEGOO_BOT_MCP_API_KEY=' {BOT_DIR}/.env | cut -d= -f2-")
    bearer = bearer.strip()
    mcp_token = fetch_mcp_token(bot)

    # restart mcp-api
    run(bot, f"cd {BOT_DIR} && printf '4\\n' | bash start.sh", timeout=180)
    code, out = run(bot, "curl -sf http://127.0.0.1:3120/health")
    print("=== Bot mcp-api health ===", out.strip())
    if code != 0:
        failed = True

    tests = [
        ("A-share flow", "/getCapitalFlow", {"mcp_token": mcp_token, "code": "600519.SH", "period": "DAY"}),
        ("A-share dist", "/getCapitalDistribution", {"mcp_token": mcp_token, "code": "600519.SH"}),
        ("HK dist", "/getCapitalDistribution", {"mcp_token": mcp_token, "code": "00700.HK"}),
    ]
    for label, path, payload in tests:
        body = json.dumps(payload, ensure_ascii=False).replace("'", "'\\''")
        cmd = (
            f"curl -s -m 90 -w '\\nHTTP %{{http_code}}\\n' "
            f"-H 'Authorization: Bearer {bearer}' -H 'Content-Type: application/json' "
            f"-d '{body}' http://127.0.0.1:3120{path}"
        )
        code, out = run(bot, cmd, timeout=120)
        print(f"\n=== Bot {label} ===\n{out[:900]}")
        if "HTTP 200" not in out:
            failed = True

    agent = ssh("geegoo-agent")
    code, out = run(
        agent,
        "export PATH=$HOME/.geegoo/bin:/usr/local/go/bin:$PATH GEEGOO_CONFIG=$HOME/.geegoo/config.json; geegoo doctor 2>&1 | tail -15",
    )
    print(f"\n=== Agent doctor ===\n{out}")
    if "全部检查通过" not in out and "ȫ" not in out:
        if "[FAIL]" in out:
            failed = True

    bot.close()
    agent.close()
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
