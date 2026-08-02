#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
bot = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(bot["host"], username=bot["user"], password=bot.get("password"))

cmds = [
    "cd /home/ubuntu/apps/GeeGooBot && git log -1 --oneline",
    "cd /home/ubuntu/apps/GeeGooBot && git fetch origin main && git reset --hard origin/main && git log -1 --oneline",
]
for cmd in cmds:
    print(">>>", cmd)
    _, o, e = c.exec_command(cmd, timeout=120)
    print((o.read()+e.read()).decode())

# rebuild app-api
print(">>> rebuild app-api")
_, o, e = c.exec_command("cd /home/ubuntu/apps/GeeGooBot && printf '3\\n' | bash start.sh 2>&1 | tail -5", timeout=180)
print((o.read()+e.read()).decode())

cmd = r"""bash -lc 'cd /home/ubuntu/apps/GeeGooBot && set -a && source .env && set +a && python3 <<PY
import json, os, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
doc = db.user_security.find_one({"code": "00700.HK"})
print("found", doc is not None)
uid = str(doc["user_id"])
key = os.environ["GEEGOO_BOT_APP_API_KEY"]
body = {"user_id": uid, "type": "flag", "frequency": "daily", "signal_index_list": ["6623e226f71be5ed2500ecfa"], "language": "cn"}
req = urllib.request.Request("http://127.0.0.1:3100/getUserStockTrend", data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
with urllib.request.urlopen(req, timeout=60) as r:
    data = json.loads(r.read().decode())
for row in data:
    if "00700" in str(row.get("code")):
        print(row)
PY'"""
print(">>> verify")
_, o, e = c.exec_command(cmd, timeout=90)
print((o.read()+e.read()).decode("utf-8", "replace"))
