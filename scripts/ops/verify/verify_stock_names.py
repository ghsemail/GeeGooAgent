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
import json, os
from pymongo import MongoClient
import urllib.request

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
doc = db.user_security.find_one()
if not doc:
    print("no user_security"); raise SystemExit
uid = str(doc["user_id"])
print("mongo raw name samples:")
for d in db.user_security.find({"user_id": doc["user_id"]}).limit(8):
    n = d.get("name")
    print(" ", d.get("code"), "name=", n)

app_key = os.environ["GEEGOO_BOT_APP_API_KEY"]
body = {"user_id": uid, "type": "flag", "frequency": "daily", "signal_index_list": ["6623e226f71be5ed2500ecfa"], "language": "cn"}
req = urllib.request.Request("http://127.0.0.1:3100/getUserStockTrend", data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+app_key})
with urllib.request.urlopen(req, timeout=60) as r:
    data = json.loads(r.read().decode())
print("api names:")
for row in data[:8]:
    print(" ", row.get("code"), "name=", repr(row.get("name")))
PY'"""
_, o, e = c.exec_command(cmd, timeout=90)
print((o.read()+e.read()).decode("utf-8", "replace"))
