#!/usr/bin/env python3
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
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

for coll, path, body in [
    ("dca_bot","getDCABotLog",None),
    ("grid_bot","getGRIDBotLog",{"hold":False}),
]:
    doc = db[coll].find_one()
    bid = str(doc["_id"])
    b = {"bot_id": bid, "hold": "all"} if coll=="dca_bot" else {"bot_id": bid, "hold": False}
    res = post(path, b)
    log = (res.get("log") or [None])[0]
    print("===", path)
    if not log: print("empty"); continue
    for k in ("time","next_opt","type"):
        v = log.get(k)
        print(k, type(v).__name__, repr(v)[:120])
    bs = log.get("buy_signal")
    print("buy_signal type", type(bs).__name__)
    if isinstance(bs, dict):
        print(" buy_signal keys", list(bs.keys())[:8])
        print(" next_opt in buy", type(bs.get("value",{}).get("next_opt") if isinstance(bs.get("value"),dict) else bs.get("next_opt")).__name__)
PY'"""
_, o, e = c.exec_command(script, timeout=90)
print((o.read()+e.read()).decode("utf-8", "replace"))
