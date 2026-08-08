#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r'''
from datetime import datetime
from bson import ObjectId
from pymongo import MongoClient

uid = ObjectId("6366170502d5c175fd586fe8")
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=",1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=",1)[1]
db = MongoClient(mongo_uri)[dbn]
today = datetime.now().strftime("%Y-%m-%d")
print("today", today)

for coll in ["stock_premarket_report", "stock_postmarket_report", "intraday_report"]:
    n_today = db[coll].count_documents({"user_id": uid, "updated_at": {"$gte": datetime.strptime(today, "%Y-%m-%d")}})
    latest = db[coll].find_one({"user_id": uid}, sort=[("updated_at", -1)])
    print(coll, "updated_today", n_today, "latest", latest.get("updated_at") if latest else None, "code", latest.get("code") if latest else None)

import subprocess
for log in ["worker.out", "service-api.out", "mcp-api.out"]:
    p = f"/home/ubuntu/apps/GeeGooBot/{log}"
    out = subprocess.run(f"tail -n 20 {p} 2>/dev/null", shell=True, capture_output=True, text=True)
    print(f"--- {log} ---")
    print(out.stdout[-1200:])
'''

cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
_, o, e = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=60)
print((o.read() + e.read()).decode("utf-8", errors="replace"))
c.close()
