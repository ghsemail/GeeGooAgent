#!/usr/bin/env python3
import json
import subprocess
import sys
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
bot = cfg["targets"]["geegoo-bot"]["ssh"]

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(bot["host"], username=bot["user"], password=bot.get("password"))

def run(cmd, timeout=180):
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", "replace")
    print(out)
    return out

print("=== git sync ===")
run("cd /home/ubuntu/apps/GeeGooBot && git fetch origin main && git reset --hard origin/main && git log -1 --oneline")

print("=== restart app-api ===")
run("cd /home/ubuntu/apps/GeeGooBot && printf '3\\n' | bash start.sh 2>&1 | tail -8")

print("=== verify names ===")
run(r"""bash -lc 'cd /home/ubuntu/apps/GeeGooBot && set -a && source .env && set +a && python3 <<PY
import json, os, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
doc = db.user_security.find_one({"code": "00700.HK"})
uid = str(doc["user_id"])
key = os.environ["GEEGOO_BOT_APP_API_KEY"]
idx = ["6623e226f71be5ed2500ecfa"]

def name_for(lang):
    body = {"user_id": uid, "type": "flag", "frequency": "daily", "signal_index_list": idx, "language": lang}
    req = urllib.request.Request("http://127.0.0.1:3100/getUserStockTrend", data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    with urllib.request.urlopen(req, timeout=60) as r:
        data = json.loads(r.read().decode())
    for row in data:
        if row.get("code") == "00700.HK":
            return row.get("name")
    return None

for lang in ("cn", "en", "hk"):
    print(lang, "->", name_for(lang))
PY'""")

c.close()
