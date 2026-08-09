#!/usr/bin/env python3
"""Why SPCX DCA had buy signal but no fill on 2026-08-08."""
from __future__ import annotations
import json
from pathlib import Path
import paramiko

REMOTE = r'''
import json
from datetime import datetime, timedelta
from bson import ObjectId
from pymongo import MongoClient

def env(k):
    for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
        if line.startswith(k + "="):
            return line.strip().split("=", 1)[1]
    return ""

db = MongoClient(env("GEEGOO_BOT_MONGO_URI"))[env("GEEGOO_BOT_MONGO_DB") or "QT_DB"]
bid = ObjectId("6a449bc14b77fe41d732b809")
bot = db.dca_bot.find_one({"_id": bid})
info = db.dca_info.find_one({"bot_id": bid})
user = db.user.find_one({"_id": ObjectId("6366170502d5c175fd586fe8")})

print("=== bot / info ===")
print("botname", bot.get("botname"), "code", bot.get("code"), "switch", bot.get("switch"))
print("dca_info status", info.get("status"), "switch", info.get("switch"))
print("position", json.dumps(info.get("position"), ensure_ascii=False, default=str))
print("user agent field", user.get("agent"), "trade_bind", user.get("trade_bind") or user.get("futu"))

att = bot.get("attitude") or {}
print("attitude switch", att.get("switch"), "controll_switch", att.get("controll_switch"), "attitude", att.get("attitude"), "date", att.get("date"))

print("\n=== DCA logs 2026-08-08 with buy next_opt ===")
for lg in db.dca_log.find({"bot_id": bid, "type": "DCA", "time": {"$regex": "^2026-08-08"}}).sort("time", 1):
    log = lg.get("log") or {}
    if log.get("next_opt") != "buy":
        continue
    pos = log.get("position") or {}
    ta = log.get("trade_agent") or {}
    print("---", lg.get("time"))
    print("  next_opt", log.get("next_opt"), "order_status", pos.get("order_status"), "order_id", pos.get("order_id"), "opt", pos.get("opt"))
    print("  trade_agent", json.dumps(ta, ensure_ascii=False)[:500])
    if log.get("notice") or log.get("message") or log.get("error"):
        print("  msg", log.get("notice") or log.get("message") or log.get("error"))

print("\n=== DCASR logs 2026-08-08 ===")
for lg in db.dca_log.find({"bot_id": bid, "type": "DCASR", "time": {"$regex": "^2026-08-08"}}).sort("time", -1).limit(10):
    log = lg.get("log") or {}
    pos = log.get("position") or {}
    print(lg.get("time"), "next", log.get("next_opt"), "opt", pos.get("opt"), "order", pos.get("order_status"), "qty", pos.get("qty"))

print("\n=== any order / fail keywords recent ===")
for lg in db.dca_log.find({"bot_id": bid}).sort("_id", -1).limit(80):
    log = lg.get("log") or {}
    pos = log.get("position") or {}
    blob = json.dumps(log, ensure_ascii=False).lower()
    if any(k in blob for k in ("fail", "error", "reject", "denied", "insufficient", "下单", "失败", "拒绝", "attitude")):
        print(lg.get("time"), lg.get("type"), log.get("next_opt"), pos.get("order_status"))
        if "attitude" in blob or "agent" in blob:
            print(" ", str(log.get("trade_agent") or log.get("attitude_gate") or log.get("notice"))[:300])

print("\n=== successful buys ever ===")
for lg in db.dca_log.find({"bot_id": bid, "log.position.opt": "buy"}).sort("_id", -1).limit(5):
    log = lg.get("log") or {}
    pos = log.get("position") or {}
    print(lg.get("time"), "opt", pos.get("opt"), "order", pos.get("order_status"), "id", pos.get("order_id"))

print("\n=== check attitude gate on buy logs ===")
for lg in db.dca_log.find({"bot_id": bid, "type": "DCA", "time": {"$regex": "^2026-08-08 03:"}}).sort("time", -1).limit(3):
    log = lg.get("log") or {}
    for k in sorted(log.keys()):
        if k in ("attitude", "attitude_gate", "attitude_result", "trade_agent", "agent_gate", "notice", "message", "error", "skip_reason"):
            print(lg.get("time"), k, str(log.get(k))[:400])
'''

def main():
    cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/probe_dca_buy3.py"
    with c.open_sftp().file(p, "w") as f: f.write(REMOTE)
    _, o, e = c.exec_command("python3 " + p, timeout=180)
    print((o.read() + e.read()).decode("utf-8", "replace"))
    c.close()

if __name__ == "__main__":
    main()
