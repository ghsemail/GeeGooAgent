#!/usr/bin/env python3
"""Probe DCA bots for SpaceX / SPCX.US and why no buy yesterday."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER_ID = "6366170502d5c175fd586fe8"
API_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
MCP = "http://118.195.135.97:3120"
TARGET_DATE = "2026-08-08"

REMOTE = r"""
import json, os, urllib.request, urllib.error
from datetime import datetime, timedelta
from bson import ObjectId
from pymongo import MongoClient

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

def mcp_post(path, body):
    req = urllib.request.Request(
        MCP + path, data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json", "Authorization": "Bearer " + API_KEY},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=90) as r:
            return json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        try:
            return json.loads(e.read().decode())
        except Exception:
            return {"raw": e.read().decode(errors="replace")[:500]}

print("=== All user bots (getUserBot via mongo dca_bot) ===")
dca_bots = list(db["dca_bot"].find({"user_id": USER_ID}))
print("dca_bot count", len(dca_bots))
for b in dca_bots:
    code = b.get("code", "")
    if "SPCX" in str(code).upper() or "SPACE" in str(b.get("stock_name", "")).upper() or "SPACE" in str(b.get("botname", "")).upper():
        print("--- DCA MATCH ---")
    print(json.dumps({
        "bot_id": str(b["_id"]),
        "botname": b.get("botname"),
        "code": b.get("code"),
        "stock_name": b.get("stock_name"),
        "frequency": b.get("frequency"),
        "switch": b.get("switch"),
        "signal": b.get("signal"),
        "order_size": b.get("order_size"),
        "price": b.get("price"),
    }, ensure_ascii=False, default=str)[:2000])

print("\n=== getAllDCABots MCP ===")
r = mcp_post("/getAllDCABots", {"mcp_token": mcp_token})
if r.get("code") == 100:
    for b in r.get("data") or []:
        code = str(b.get("code", ""))
        name = str(b.get("botname", "")) + str(b.get("stock_name", ""))
        if "SPCX" in code.upper() or "SPACE" in name.upper():
            print("DCA", json.dumps({k: b.get(k) for k in [
                "bot_id", "botname", "code", "stock_name", "bot_switch", "status", "frequency",
                "signal", "order_size", "price", "position", "attitude"
            ]}, ensure_ascii=False, default=str)[:3000])
else:
    print(r)

print("\n=== get_report_bot_codes (workflow list) ===")
r = mcp_post("/getReportBotCodes", {"mcp_token": mcp_token})
if r.get("code") == 100:
    for b in r.get("data") or []:
        if "SPCX" in str(b.get("code", "")).upper() or "SPACE" in str(b.get("stock_name", "")).upper():
            print("report_bot", b)

print("\n=== DCA logs for SPCX-related bots ===")
start = datetime.strptime(TARGET_DATE, "%Y-%m-%d")
end = start + timedelta(days=2)
for b in dca_bots:
    code = str(b.get("code", ""))
    name = str(b.get("botname", "")) + str(b.get("stock_name", ""))
    if "SPCX" not in code.upper() and "SPACE" not in name.upper():
        continue
    bid = str(b["_id"])
    print("\nBOT", bid, b.get("botname"), code)
    info = db["dca_info"].find_one({"bot_id": bid})
    if info:
        print("dca_info status/switch:", info.get("status"), info.get("switch"), "position", info.get("position"))
    r = mcp_post("/getDCABotLog", {"mcp_token": mcp_token, "bot_id": bid, "hold": False, "filter": "all"})
    if r.get("code") == 100:
        data = r.get("data") or r
        logs = data.get("log") if isinstance(data, dict) else None
        if logs is None and isinstance(r.get("log"), list):
            logs = r.get("log")
        if isinstance(data, dict) and not logs:
            logs = data.get("log_sr")
        print("log wrapper keys", list(data.keys()) if isinstance(data, dict) else type(data))
        if isinstance(logs, list):
            print("total logs", len(logs))
            for row in logs[:25]:
                t = row.get("time")
                sig = row.get("signal") or row.get("buy_signal") or {}
                pos = row.get("position") or {}
                print(t, "next_opt=", row.get("next_opt"), "opt=", pos.get("opt"), "order_status=", pos.get("order_status"), "signal_buy=", sig.get("buy") if isinstance(sig, dict) else sig)
    else:
        print("getDCABotLog err", r)
    rows = list(db["dca_log"].find({"bot_id": bid, "time": {"$gte": start, "$lt": end}}).sort("time", 1))
    print("dca_log rows on target date", len(rows))
    for row in rows[:30]:
        lg = row.get("log") or {}
        pos = lg.get("position") or {}
        print(" ", row.get("time"), "next_opt=", lg.get("next_opt"), "opt=", pos.get("opt"), "order=", pos.get("order_status"))
        if lg.get("signal") or lg.get("reason"):
            print("   ", str(lg.get("signal") or lg.get("reason"))[:300])

print("\n=== All DCA bots summary ===")
for b in dca_bots:
    bid = str(b["_id"])
    info = db["dca_info"].find_one({"bot_id": bid}) or {}
    print(b.get("code"), b.get("botname"), "switch=", b.get("switch"), "status=", info.get("status"), "bot_id=", bid[:12])
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    remote_path = "/tmp/probe_spacex_dca.py"
    sftp = c.open_sftp()
    with sftp.file(remote_path, "w") as f:
        f.write(REMOTE)
    sftp.close()
    _, o, e = c.exec_command(f"python3 {remote_path}", timeout=240)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
