#!/usr/bin/env python3
"""Run A-share GRID intraday decision: pick real GRID bot for 601766.SH."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER_ID = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
TARGET_CODE = "601766.SH"


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


def pick_cn_grid_bot_from_mcp(mcp_token: str) -> dict:
    remote = f"""
import json, urllib.request
mcp_token = {json.dumps(mcp_token)}
api_key = {json.dumps(BOT_KEY)}
req = urllib.request.Request(
    "http://118.195.135.97:3120/getReportBotCodes",
    data=json.dumps({{"mcp_token": mcp_token}}).encode(),
    headers={{"Content-Type": "application/json", "Authorization": "Bearer " + api_key}},
    method="POST",
)
data = json.loads(urllib.request.urlopen(req, timeout=60).read())
rows = data.get("data") or []
pick = None
for row in rows:
    code = str(row.get("code") or "")
    if "601766" in code:
        pick = row
        break
if pick is None:
    for row in rows:
        code = str(row.get("code") or "")
        if code.endswith(".SH") and row.get("bot_type") in ("GRID", "DCA"):
            pick = row
            break
print(json.dumps(pick, ensure_ascii=False, default=str))
"""
    lines = [ln.strip() for ln in ssh_run("geegoo-agent", f"python3 <<'PY'\n{remote}\nPY", timeout=90).splitlines() if ln.strip()]
    raw = lines[-1] if lines else ""
    if raw in ("null", "", "None"):
        raise RuntimeError(f"no bot found for {TARGET_CODE} via getReportBotCodes")
    bot = json.loads(raw)
    return {
        "code": bot.get("code") or TARGET_CODE,
        "stock_name": bot.get("stock_name") or "中国中车",
        "bot_id": str(bot.get("bot_id")),
        "bot_name": bot.get("bot_name") or bot.get("botname") or "",
        "bot_type": bot.get("bot_type") or "GRID",
        "frequency": "15m",
        "trade_type": "信号买入",
        "attitude_switch": True,
    }


def pick_cn_grid_bot(mcp_token: str) -> dict:
    remote = f"""
import json
from bson import ObjectId
from pymongo import MongoClient

USER_ID = ObjectId({json.dumps(USER_ID)})
TARGET = {json.dumps(TARGET_CODE)}
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=", 1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=", 1)[1]
db = MongoClient(mongo_uri)[dbn]
rows = []
for doc in db["grid_bot"].find({{"user_id": USER_ID}}):
    code = str(doc.get("code") or "")
    if "601766" not in code:
        continue
    att = doc.get("attitude") or {{}}
    rows.append({{
        "bot_id": str(doc["_id"]),
        "botname": doc.get("botname") or doc.get("bot_name") or "",
        "code": code,
        "stock_name": doc.get("stock_name") or "中国中车",
        "frequency": doc.get("frequency") or "15m",
        "attitude": att,
    }})
if not rows:
    for doc in db["dca_bot"].find({{"user_id": USER_ID}}):
        code = str(doc.get("code") or "")
        if "601766" not in code:
            continue
        att = doc.get("attitude") or {{}}
        rows.append({{
            "bot_id": str(doc["_id"]),
            "botname": doc.get("botname") or doc.get("bot_name") or "",
            "code": code,
            "stock_name": doc.get("stock_name") or "中国中车",
            "frequency": doc.get("frequency") or "15m",
            "attitude": att,
            "bot_type": "DCA",
        }})
prefer = [r for r in rows if r.get("code", "").endswith(".SH")]
pick = (prefer or rows or [None])[0]
print(json.dumps(pick, ensure_ascii=False, default=str))
"""
    lines = [ln.strip() for ln in ssh_run("geegoo-bot", f"python3 <<'PY'\n{remote}\nPY", timeout=90).splitlines() if ln.strip()]
    raw = lines[-1] if lines else ""
    if raw in ("null", "", "None"):
        return pick_cn_grid_bot_from_mcp(mcp_token)
    bot = json.loads(raw)
    att = bot.get("attitude") or {}
    return {
        "code": bot.get("code") or TARGET_CODE,
        "stock_name": bot.get("stock_name") or "中国中车",
        "bot_id": str(bot.get("bot_id")),
        "bot_name": bot.get("botname") or bot.get("bot_name") or bot.get("stock_name", "中国中车") + "机器人",
        "bot_type": bot.get("bot_type") or "GRID",
        "frequency": str(bot.get("frequency") or "15m"),
        "trade_type": "信号买入",
        "attitude_switch": bool(att.get("switch")),
    }


def delete_intraday_reports(mcp_token: str, code: str, bot_id: str) -> str:
    remote = f"""
import json, urllib.request
mcp_token = {json.dumps(mcp_token)}
api_key = {json.dumps(BOT_KEY)}
code = {json.dumps(code)}
bot_id = {json.dumps(bot_id)}
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

rows = post("/getStockIntradayReports", {{"mcp_token": mcp_token, "code": code, "bot_id": bot_id, "limit": 50}}).get("data") or []
print("delete_before_count", len(rows))
for row in rows:
    rid = row.get("report_id")
    if not rid:
        continue
    res = post("/deleteStockIntradayReport", {{"mcp_token": mcp_token, "report_id": rid}})
    print("deleted", rid, res.get("code"))
rows2 = post("/getStockIntradayReports", {{"mcp_token": mcp_token, "code": code, "bot_id": bot_id, "limit": 50}}).get("data") or []
print("delete_after_count", len(rows2))
"""
    return ssh_run("geegoo-agent", f"python3 <<'PY'\n{remote}\nPY", timeout=120)


def run_intraday(mcp_token: str, sim: dict) -> str:
    intraday = {
        "code": sim["code"],
        "stock_name": sim["stock_name"],
        "bot_id": sim["bot_id"],
        "bot_name": sim["bot_name"],
        "bot_type": sim["bot_type"],
        "frequency": sim["frequency"],
        "trade_type": sim["trade_type"],
        "attitude_switch": sim["attitude_switch"],
    }
    body_json = json.dumps(
        {"skill": "intraday_stock", "mcp_token": mcp_token, "intraday": intraday},
        ensure_ascii=False,
    )
    remote = f"""
import json, urllib.request, time
BODY = json.loads({json.dumps(body_json)})
req = urllib.request.Request(
    "http://127.0.0.1:3400/v1/skills/run",
    data=json.dumps(BODY, ensure_ascii=False).encode(),
    headers={{"Content-Type": "application/json"}},
    method="POST",
)
t0 = time.time()
with urllib.request.urlopen(req, timeout=600) as r:
    print("elapsed_sec", round(time.time()-t0, 1))
    print("http", r.status)
    print(r.read().decode())
"""
    return ssh_run("geegoo-agent", f"python3 <<'PY'\n{remote}\nPY", timeout=620)


def verify_report(mcp_token: str, sim: dict) -> str:
    remote = f"""
import json, urllib.request
mcp_token = {json.dumps(mcp_token)}
api_key = {json.dumps(BOT_KEY)}
body = {{"mcp_token": mcp_token, "code": {json.dumps(sim["code"])}, "bot_id": {json.dumps(sim["bot_id"])}, "limit": 3}}
req = urllib.request.Request(
    "http://118.195.135.97:3120/getStockIntradayReports",
    data=json.dumps(body).encode(),
    headers={{"Content-Type": "application/json", "Authorization": "Bearer " + api_key}},
    method="POST",
)
data = json.loads(urllib.request.urlopen(req, timeout=60).read())
rows = data.get("data") or []
print("api_code", data.get("code"), "count", len(rows))
if rows:
    latest = rows[0]
    print(json.dumps({{k: latest.get(k) for k in [
        "report_id", "code", "stock_name", "bot_name", "bot_type", "result", "confidence",
        "trade_type", "price", "summary", "reason"
    ]}}, ensure_ascii=False, default=str))
    rep = str(latest.get("report") or "")
    print("report_len", len(rep))
    print(rep[:3500])
else:
    print("no intraday reports")
"""
    return ssh_run("geegoo-agent", f"python3 <<'PY'\n{remote}\nPY", timeout=90)


def main() -> int:
    print("=== CN GRID intraday live ===")
    mcp_token = fetch_mcp_token()
    print("mcp_token_prefix", mcp_token[:16] + "…")
    sim = pick_cn_grid_bot(mcp_token)
    print("picked_bot", json.dumps(sim, ensure_ascii=False))
    print("\n=== delete previous intraday reports ===")
    print(delete_intraday_reports(mcp_token, sim["code"], sim["bot_id"]))
    print("\n=== POST /v1/skills/run ===")
    print(run_intraday(mcp_token, sim))
    print("\n=== getStockIntradayReports ===")
    print(verify_report(mcp_token, sim))
    return 0


if __name__ == "__main__":
    sys.exit(main())
