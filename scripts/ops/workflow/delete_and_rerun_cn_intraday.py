#!/usr/bin/env python3
"""Delete simulated A-share intraday reports and regenerate one fresh report."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER_ID = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
SIM = {
    "code": "601766.SH",
    "stock_name": "中国中车",
    "bot_id": "6781cb8309a2189f26d8866e",
    "bot_name": "A股盘中模拟",
    "bot_type": "GRID",
    "frequency": "15m",
    "trade_type": "信号买入",
}


def ssh_run(target: str, cmd: str, timeout: int = 900) -> str:
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


def delete_and_regenerate(mcp_token: str) -> str:
    remote = f"""
import json, urllib.request, time

MCP = {json.dumps(mcp_token)}
API = {json.dumps(BOT_KEY)}
CODE = {json.dumps(SIM["code"])}
BOT_ID = {json.dumps(SIM["bot_id"])}
BASE = "http://118.195.135.97:3120"

def post(path, body):
    req = urllib.request.Request(
        BASE + path,
        data=json.dumps(body).encode(),
        headers={{"Content-Type": "application/json", "Authorization": "Bearer " + API}},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read())

rows = post("/getStockIntradayReports", {{"mcp_token": MCP, "code": CODE, "bot_id": BOT_ID, "limit": 50}}).get("data") or []
print("before_count", len(rows))
for row in rows:
    rid = row.get("report_id")
    if not rid:
        continue
    res = post("/deleteStockIntradayReport", {{"mcp_token": MCP, "report_id": rid}})
    print("deleted", rid, res.get("code"), res.get("message"))

rows2 = post("/getStockIntradayReports", {{"mcp_token": MCP, "code": CODE, "bot_id": BOT_ID, "limit": 50}}).get("data") or []
print("after_delete_count", len(rows2))

BODY = {{
    "skill": "intraday_stock",
    "mcp_token": MCP,
    "intraday": {{
        "code": {json.dumps(SIM["code"])},
        "stock_name": {json.dumps(SIM["stock_name"])},
        "bot_id": {json.dumps(SIM["bot_id"])},
        "bot_name": {json.dumps(SIM["bot_name"])},
        "bot_type": {json.dumps(SIM["bot_type"])},
        "frequency": {json.dumps(SIM["frequency"])},
        "trade_type": {json.dumps(SIM["trade_type"])},
    }},
}}
req = urllib.request.Request(
    "http://127.0.0.1:3400/v1/skills/run",
    data=json.dumps(BODY, ensure_ascii=False).encode(),
    headers={{"Content-Type": "application/json"}},
    method="POST",
)
t0 = time.time()
with urllib.request.urlopen(req, timeout=600) as r:
    raw = r.read().decode()
    print("run_elapsed_sec", round(time.time()-t0, 1))
    print("run_http", r.status)
    print(raw)

rows3 = post("/getStockIntradayReports", {{"mcp_token": MCP, "code": CODE, "bot_id": BOT_ID, "limit": 5}}).get("data") or []
print("final_count", len(rows3))
if rows3:
    latest = rows3[0]
    print(json.dumps({{k: latest.get(k) for k in [
        "report_id", "code", "bot_name", "result", "confidence",
        "trade_type", "price", "summary", "reason"
    ]}}, ensure_ascii=False))
    rep = str(latest.get("report") or "")
    print("report_len", len(rep))
    print(rep[:2500])
else:
    print("no report after regenerate")
"""
    return ssh_run("geegoo-agent", f"python3 <<'PY'\n{remote}\nPY", timeout=620)


def main() -> int:
    print("=== Delete simulated A-share intraday + regenerate ===")
    print(json.dumps(SIM, ensure_ascii=False))
    mcp_token = fetch_mcp_token()
    print("mcp_token_prefix", mcp_token[:16] + "…")
    print(delete_and_regenerate(mcp_token))
    return 0


if __name__ == "__main__":
    sys.exit(main())
