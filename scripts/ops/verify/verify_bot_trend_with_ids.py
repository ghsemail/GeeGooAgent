#!/usr/bin/env python3
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

bot_db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
doc = bot_db.user_security.find_one({"code": {"$regex": "\\.HK$"}})
if not doc:
    doc = bot_db.user_security.find_one()
uid = str(doc["user_id"])
app_key = os.environ["GEEGOO_BOT_APP_API_KEY"]
idx_ids = ["6623e226f71be5ed2500ecfa", "662492aa585ef3df59f8bb8d", "662c9459c4cee7ffb800d0a3"]

def call(path, body):
    req = urllib.request.Request("http://127.0.0.1:3100/"+path, data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+app_key})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

trends = call("getUserStockTrend", {"user_id": uid, "type": "flag", "frequency": "daily", "signal_index_list": idx_ids, "language": "cn"})
print("getUserStockTrend with index ids:")
for row in trends[:6]:
    print(" ", row.get("code"), row.get("trend"))

grid = bot_db.grid_bot.find_one({"user_id": doc["user_id"]})
if grid:
    logs = call("getGRIDBotLog", {"bot_id": str(grid["_id"]), "hold": False})
    print("getGRIDBotLog code=", logs.get("code"), "log_len=", len(logs.get("log") or []), "has_info=", "info" in logs)
PY'"""
_, o, e = c.exec_command(cmd, timeout=90)
print((o.read()+e.read()).decode("utf-8", "replace"))
