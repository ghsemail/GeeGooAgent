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
    "cd /home/ubuntu/apps/GeeGooBot && git fetch origin main && git reset --hard origin/main && git log -1 --oneline",
    "cd /home/ubuntu/apps/GeeGooBot && printf '3\\n' | bash start.sh 2>&1 | tail -5",
]
for cmd in cmds:
    print(">>>", cmd)
    _, o, e = c.exec_command(cmd, timeout=180)
    print((o.read()+e.read()).decode())

script = r"""bash -lc 'cd /home/ubuntu/apps/GeeGooBot && set -a && source .env && set +a && python3 <<PY
import json, os, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
key = os.environ["GEEGOO_BOT_APP_API_KEY"]
rem = db.dca_reminder.find_one()
if not rem:
    print("no dca_reminder"); raise SystemExit
rid = str(rem["_id"])

def post(path, body):
    req = urllib.request.Request("http://127.0.0.1:3100/"+path, data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

res = post("getDCAReminderLog", {"bot_id": rid, "hold": False})
print("type", type(res).__name__, "len", len(res) if isinstance(res, list) else res)
if isinstance(res, list) and res:
    print("first_keys", list(res[0].keys()))
PY'"""
print(">>> verify")
_, o, e = c.exec_command(script, timeout=90)
print((o.read()+e.read()).decode("utf-8", "replace"))
