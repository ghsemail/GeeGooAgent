#!/usr/bin/env python3
"""Delete all intraday decision reports for ghsemail user."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER_ID = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def ssh_run(target: str, cmd: str, timeout: int = 180) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def fetch_mcp_token() -> str:
    remote = f"""
from bson import ObjectId
from pymongo import MongoClient

USER_ID = ObjectId({json.dumps(USER_ID)})
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=", 1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=", 1)[1]
user = MongoClient(mongo_uri)[dbn]["user"].find_one({{"_id": USER_ID}})
print((user or {{}}).get("mcp", {{}}).get("mcp_token", ""))
"""
    token = ssh_run("geegoo-bot", f"python3 <<'PY'\n{remote}\nPY", timeout=60).strip().splitlines()[-1].strip()
    if not token.startswith("mcp_"):
        raise RuntimeError(f"mcp_token lookup failed: {token!r}")
    return token


def clear_all(mcp_token: str) -> str:
    remote = f"""
import json, urllib.request
mcp_token = {json.dumps(mcp_token)}
api_key = {json.dumps(BOT_KEY)}
BASE = "http://118.195.135.97:3120"

def post(path, body):
    req = urllib.request.Request(
        BASE + path,
        data=json.dumps(body).encode(),
        headers={{"Content-Type": "application/json", "Authorization": "Bearer " + api_key}},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read())

rows = post("/getStockIntradayReports", {{"mcp_token": mcp_token, "limit": 500}}).get("data") or []
print("found", len(rows))
for row in rows:
    rid = row.get("report_id")
    if not rid:
        continue
    code = row.get("code")
    bot = row.get("bot_name")
    res = post("/deleteStockIntradayReport", {{"mcp_token": mcp_token, "report_id": rid}})
    print("deleted", rid, code, bot, res.get("code"))
left = post("/getStockIntradayReports", {{"mcp_token": mcp_token, "limit": 500}}).get("data") or []
print("remaining", len(left))
"""
    return ssh_run("geegoo-agent", f"python3 <<'PY'\n{remote}\nPY", timeout=180)


def main() -> int:
    print("=== clear all intraday reports ===")
    mcp_token = fetch_mcp_token()
    print("mcp_token_prefix", mcp_token[:16] + "…")
    print(clear_all(mcp_token))
    return 0


if __name__ == "__main__":
    sys.exit(main())
