#!/usr/bin/env python3
"""Probe all bot log/profit APIs on production."""
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
bot = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(bot["host"], username=bot["user"], password=bot.get("password"))

script = r"""bash -lc 'cd /home/ubuntu/apps/GeeGooBot && set -a && source .env && set +a && python3 <<PY
import json, os, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
key = os.environ["GEEGOO_BOT_APP_API_KEY"]

def post(path, body):
    req = urllib.request.Request("http://127.0.0.1:3100/"+path, data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            raw = r.read().decode()
            return r.status, json.loads(raw)
    except Exception as e:
        err = e.read().decode() if hasattr(e,"read") else str(e)
        return getattr(e,"code",0), err

checks = [
    ("grid_bot", "getGRIDBotLog", lambda id: {"bot_id": id, "hold": False}),
    ("grid_bot", "getGRIDBotProfit", lambda id: {"bot_id": id}),
    ("dca_bot", "getDCABotLog", lambda id: {"bot_id": id, "hold": False, "filter": "all"}),
    ("dca_bot", "getDCABotProfit", lambda id: {"bot_id": id}),
    ("hdg_bot", "getHDGBotLog", lambda id: {"bot_id": id}),
    ("hdg_bot", "getHDGBotProfit", lambda id: {"bot_id": id}),
    ("trade_bot", "getSmartTradeLog", lambda id: {"bot_id": id, "hold": False}),
    ("trade_bot", "getSmartTradeProfit", lambda id: {"bot_id": id}),
]

for coll, path, body_fn in checks:
    doc = db[coll].find_one()
    if not doc:
        print(coll, path, "NO_BOT")
        continue
    bid = str(doc["_id"])
    st, res = post(path, body_fn(bid))
    if isinstance(res, dict) and res.get("code") not in (None, 100):
        print(path, "ERROR", res)
        continue
    if isinstance(res, list):
        print(path, "list", len(res))
    elif isinstance(res, dict):
        logs = res.get("log") or res.get("log_sr")
        pl = res.get("profit_list")
        tp = res.get("total_profit")
        print(path, "dict keys", list(res.keys()), "log_len", len(logs) if isinstance(logs,list) else None, "profit_list", len(pl) if isinstance(pl,list) else None, "total_profit", tp)
        if isinstance(logs, list) and logs:
            print("  first_log", list(logs[0].keys())[:12])
            pos = logs[0].get("position") or {}
            for k,v in list(pos.items())[:6]:
                print("   pos", k, type(v).__name__, v)
        if isinstance(pl, list) and pl:
            print("  first_profit", pl[0])
PY'"""
_, o, e = c.exec_command(script, timeout=180)
print((o.read()+e.read()).decode("utf-8", "replace"))
