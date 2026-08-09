#!/usr/bin/env python3
"""Deploy GeeGooAgent and end-to-end verify capital routing."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
AGENT_DIR = "/home/ubuntu/.geegoo/geegoo-agent"
BOT_DIR = "/home/ubuntu/apps/GeeGooBot"
OUT = Path(r"D:\Geegoo\GeeGooAgent\.tmp\deploy_verify.txt")


def load(name: str) -> dict:
    return json.loads(DEPLOY.read_text(encoding="utf-8-sig"))["targets"][name]["ssh"]


def run(c: paramiko.SSHClient, cmd: str, timeout: int = 300) -> tuple[int, str]:
    _, o, e = c.exec_command(cmd, get_pty=True, timeout=timeout)
    text = (o.read() + e.read()).decode("utf-8", errors="replace")
    if len(text) > 12000:
        text = text[:6000] + "\n...(truncated)...\n" + text[-5000:]
    return o.channel.recv_exit_status(), text


def connect(name: str) -> paramiko.SSHClient:
    ssh = load(name)
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=60)
    return c


def main() -> int:
    lines: list[str] = []
    failed = False

    def step(title: str, code: int, out: str) -> None:
        nonlocal failed
        lines.append(f"\n=== {title} ===\n{out}\n")
        if code != 0:
            failed = True
            lines.append(f"[FAIL exit {code}]\n")

    # 1) Bot: sync already done; ensure mcp-api up
    bot = connect("geegoo-bot")
    step(
        "Bot mcp-api restart",
        *run(
            bot,
            f"cd {BOT_DIR} && export PATH=/usr/local/go/bin:$PATH GOFLAGS=-mod=vendor && "
            f"go build -o bin/mcpAPIServer ./cmd/mcp-api && "
            f"pkill -f mcpAPIServer || true; sleep 1; "
            f"set -a && source .env && set +a && nohup ./bin/mcpAPIServer >> mcp-api.out 2>&1 & sleep 3 && "
            f"curl -sf http://127.0.0.1:3120/health",
        ),
    )

    # 2) Fetch bearer + mcp_token from mongo
    _, bearer_out = run(bot, f"grep '^GEEGOO_BOT_MCP_API_KEY=' {BOT_DIR}/.env | cut -d= -f2-")
    bearer = bearer_out.strip()
    mongo_cmd = rf"""
cd {BOT_DIR} && set -a && source .env && set +a
python3 - <<'PY'
import os, json
from urllib.parse import quote_plus
try:
    from pymongo import MongoClient
except ImportError:
    import subprocess
    subprocess.check_call([sys.executable, '-m', 'pip', 'install', 'pymongo', '-q'])
    from pymongo import MongoClient
import sys
uri = os.environ.get('GEEGOO_BOT_MONGO_URI','')
dbn = os.environ.get('GEEGOO_BOT_MONGO_DB','QT_DB')
c = MongoClient(uri, serverSelectionTimeoutMS=8000)
doc = c[dbn]['user'].find_one({{'mcp.mcp_token': {{'$exists': True, '$ne': ''}}}}, {{'mcp.mcp_token': 1}})
print(doc.get('mcp', {{}}).get('mcp_token','') if doc else '')
PY
"""
    # fix script - sys not imported in pip branch
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
    code, mcp_out = run(bot, mongo_cmd, timeout=120)
    mcp_token = mcp_out.strip().splitlines()[-1] if mcp_out.strip() else ""
    lines.append(f"\n=== Mongo mcp_token ===\nfound={'yes' if mcp_token else 'no'}\n")

    if mcp_token and bearer:
        for label, path, payload in [
            (
                "A-share capital flow (600519.SH)",
                "/getCapitalFlow",
                {"mcp_token": mcp_token, "code": "600519.SH", "period": "DAY"},
            ),
            (
                "A-share capital distribution (600519.SH)",
                "/getCapitalDistribution",
                {"mcp_token": mcp_token, "code": "600519.SH"},
            ),
            (
                "HK capital distribution (00700.HK)",
                "/getCapitalDistribution",
                {"mcp_token": mcp_token, "code": "00700.HK"},
            ),
        ]:
            body = json.dumps(payload, ensure_ascii=False)
            curl = (
                f"curl -s -m 90 -w '\\nHTTP %{{http_code}}\\n' "
                f"-H 'Authorization: Bearer {bearer}' -H 'Content-Type: application/json' "
                f"-d '{body}' http://127.0.0.1:3120{path}"
            )
            step(f"Bot {label}", *run(bot, curl, timeout=120))
    else:
        failed = True
        lines.append("[FAIL] missing bearer or mcp_token for Bot API test\n")

    # 3) CN Data node direct
    cn = connect("geegoo-data-cn")
    _, tok_out = run(cn, f"grep '^GEEGOO_DATA_SERVICE_TOKEN=' {BOT_DIR.replace('/apps/GeeGooBot', '/apps/GeeGooData')}/.env 2>/dev/null || grep '^GEEGOO_DATA_SERVICE_TOKEN=' /home/ubuntu/apps/GeeGooData/.env | cut -d= -f2-")
    cn_token = tok_out.strip()
    if not cn_token:
        lines.append(f"[FAIL] missing GEEGOO_DATA_SERVICE_TOKEN on CN data node\n")
        cn_token = ""
    if cn_token:
        step(
            "CN Data quote direct",
            *run(
                cn,
                f"curl -s -m 30 -H 'Authorization: Bearer {cn_token}' -H 'Content-Type: application/json' "
                f"-d '{{\"code\":\"600519.SH\"}}' http://127.0.0.1:3300/v1/market/quote",
            ),
        )
    else:
        failed = True
    cn.close()

    # 4) Agent deploy
    agent = connect("geegoo-agent")
    step(
        "Agent git pull + build",
        *run(
            agent,
            f"cd {AGENT_DIR} && git fetch origin main && git reset --hard origin/main && "
            f"export PATH=/usr/local/go/bin:$PATH && go build -o geegoo ./cmd/geegoo && "
            f"ln -sf {AGENT_DIR}/geegoo ~/.geegoo/bin/geegoo",
            timeout=600,
        ),
    )
    step(
        "Agent doctor",
        *run(
            agent,
            f"export PATH=$HOME/.geegoo/bin:/usr/local/go/bin:$PATH && "
            f"export GEEGOO_CONFIG=$HOME/.geegoo/config.json && "
            f"geegoo doctor 2>&1 | tail -40",
            timeout=120,
        ),
    )
    agent.close()
    bot.close()

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text("".join(lines), encoding="utf-8")
    print(OUT.read_text(encoding="utf-8"))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
