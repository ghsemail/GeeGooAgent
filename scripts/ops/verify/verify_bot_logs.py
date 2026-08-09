#!/usr/bin/env python3
"""Probe bot log API responses for parsing issues."""
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
bot = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(bot["host"], username=bot["user"], password=bot.get("password"))

cmd = r"""bash -lc 'cd /home/ubuntu/apps/GeeGooBot && set -a && source .env && set +a && python3 <<PY
import json, os, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
key = os.environ["GEEGOO_BOT_APP_API_KEY"]

def post(path, body):
    req = urllib.request.Request("http://127.0.0.1:3100/"+path, data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

for coll, path in [("grid_bot","getGRIDBotLog"),("dca_bot","getDCABotLog"),("hdg_bot","getHDGBotLog")]:
    doc = db[coll].find_one()
    if not doc:
        print(coll, "no bot"); continue
    bid = str(doc["_id"])
    print("\n===", path, "bot", bid, "===")
    try:
        if path == "getDCABotLog":
            res = post(path, {"bot_id": bid, "hold": False})
        else:
            res = post(path, {"bot_id": bid, "hold": False} if path=="getGRIDBotLog" else {"bot_id": bid})
        if isinstance(res, dict) and res.get("code") not in (None, 100):
            print("ERROR envelope", res)
            continue
        if path == "getGRIDBotLog":
            logs = res.get("log") or []
            info = res.get("info") or {}
            print("log_count", len(logs), "info", info)
            if logs:
                row = logs[0]
                print("first_log keys", list(row.keys()))
                pos = row.get("position") or {}
                print("position", pos)
                for k,v in pos.items():
                    print(" ", k, type(v).__name__, v)
            profit = post("getGRIDBotProfit", {"bot_id": bid})
            print("profit keys", list(profit.keys()) if isinstance(profit, dict) else type(profit))
            if isinstance(profit, dict):
                print("total_profit", profit.get("total_profit"))
                pl = profit.get("profit_list") or []
                if pl:
                    print("first profit row", pl[0])
                    for k,v in pl[0].items():
                        print(" ", k, type(v).__name__, v)
        else:
            print(json.dumps(res, ensure_ascii=False)[:800])
    except Exception as e:
        err = e.read().decode() if hasattr(e,"read") else str(e)
        print("FAIL", err[:400])
PY'"""
_, o, e = c.exec_command(cmd, timeout=120)
print((o.read()+e.read()).decode("utf-8", "replace"))
