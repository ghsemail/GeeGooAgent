#!/usr/bin/env python3
"""Diagnose why SpaceX SmartTrade (SPCX.US) did not buy on 2026-08-08."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
BOT_ID = "6a3ab6efcfb30b81379bf91d"
CODE = "SPCX.US"
USER_ID = "6366170502d5c175fd586fe8"
API_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
MCP = "http://118.195.135.97:3120"
TARGET_DATE = "2026-08-08"

REMOTE = r"""
import json, os, urllib.request, urllib.error
from datetime import datetime, timedelta
from bson import ObjectId
from pymongo import MongoClient

BOT_ID = """ + json.dumps(BOT_ID) + r"""
CODE = """ + json.dumps(CODE) + r"""
USER_ID = """ + json.dumps(USER_ID) + r"""
API_KEY = """ + json.dumps(API_KEY) + r"""
MCP = """ + json.dumps(MCP) + r"""
TARGET_DATE = """ + json.dumps(TARGET_DATE) + r"""

os.chdir("/home/ubuntu/apps/GeeGooBot")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
user = db["user"].find_one({"_id": ObjectId(USER_ID)})
mcp_token = (user or {}).get("mcp", {}).get("mcp_token", "")
print("mcp_token", (mcp_token[:12] + "...") if mcp_token else "MISSING")

def mcp_post(path, body):
    req = urllib.request.Request(
        MCP + path,
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + API_KEY},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=90) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors="replace")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw[:800]}

print("\n=== 1. SmartTrade config ===")
_, r = mcp_post("/getAllSmartTrades", {"mcp_token": mcp_token, "code": CODE})
bots = (r.get("data") or []) if r.get("code") == 100 else []
bot = next((b for b in bots if str(b.get("bot_id")) == BOT_ID), bots[0] if bots else None)
if bot:
    keys = ["bot_id","botname","code","bot_switch","status","trade_mode","frequency","price","order_size","position","attitude","tp","sl","binding_bot_id"]
    print(json.dumps({k: bot.get(k) for k in keys}, ensure_ascii=False, indent=2, default=str))
else:
    print("NO BOT", r)

print("\n=== 2. Mongo trade_bot / trade_info ===")
tb = db["trade_bot"].find_one({"_id": ObjectId(BOT_ID)})
ti = db["trade_info"].find_one({"bot_id": BOT_ID})
if tb:
    print("trade_bot signal:", json.dumps(tb.get("signal"), ensure_ascii=False)[:1500] if tb.get("signal") else "NONE")
    print("trade_bot attitude:", json.dumps(tb.get("attitude"), ensure_ascii=False)[:1500] if tb.get("attitude") else "NONE")
    print("trade_bot price/order:", tb.get("price"), tb.get("order_size"), "trade_mode", tb.get("trade_mode"))
if ti:
    print("trade_info status/switch:", ti.get("status"), ti.get("switch"))
    print("trade_info position:", json.dumps(ti.get("position"), ensure_ascii=False, default=str))
    print("trade_info keys sample:", [k for k in ti.keys() if k not in ("_id",)])

print("\n=== 3. Trade connection & position ===")
for ep, body in [("/checkTradeConnection", {"mcp_token": mcp_token}), ("/getPosition", {"mcp_token": mcp_token, "code": CODE})]:
    st, resp = mcp_post(ep, body)
    print(ep, st, json.dumps(resp, ensure_ascii=False)[:1200])

print("\n=== 4. SmartTrade log ===")
_, r = mcp_post("/getSmartTradeLog", {"mcp_token": mcp_token, "bot_id": BOT_ID})
if r.get("code") == 100:
    data = r.get("data") or {}
    print("info", data.get("info"))
    logs = data.get("log") or []
    print("log_count", len(logs))
    for row in logs[:20]:
        pos = row.get("position") or {}
        print(row.get("time"), "next_opt=", row.get("next_opt"), "opt=", pos.get("opt"), "order_status=", pos.get("order_status"), "qty=", pos.get("qty"))
else:
    print(r)

print("\n=== 5. trade_log on", TARGET_DATE, "===")
start = datetime.strptime(TARGET_DATE, "%Y-%m-%d")
end = start + timedelta(days=2)
rows = list(db["trade_log"].find({"bot_id": BOT_ID, "time": {"$gte": start, "$lt": end}}).sort("time", 1))
print("rows", len(rows))
for row in rows:
    lg = row.get("log") or {}
    pos = lg.get("position") or {}
    line = f"{row.get('time')} next_opt={lg.get('next_opt')} opt={pos.get('opt')} order_status={pos.get('order_status')} qty={pos.get('qty')}"
    print(line)
    for k in ("signal", "buy_signal", "reason", "message", "attitude", "analysis", "notice"):
        if lg.get(k):
            print(" ", k, str(lg.get(k))[:400])

print("\n=== 6. getBotSignal ===")
_, r = mcp_post("/getBotSignal", {"mcp_token": mcp_token, "bot_id": BOT_ID, "code": CODE})
print(json.dumps(r, ensure_ascii=False)[:3000])

print("\n=== 7. getBotLogByType ===")
_, r = mcp_post("/getBotLogByType", {"mcp_token": mcp_token, "bot_id": BOT_ID, "type": "SmartTrade"})
print(json.dumps(r, ensure_ascii=False)[:2500])
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    sftp = c.open_sftp()
    remote_path = "/tmp/probe_spacex_buy.py"
    with sftp.file(remote_path, "w") as f:
        f.write(REMOTE)
    sftp.close()
    _, o, e = c.exec_command(f"cd /home/ubuntu/apps/GeeGooBot && python3 {remote_path}", timeout=240)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
