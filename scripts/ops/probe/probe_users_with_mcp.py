#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
bc = paramiko.SSHClient()
bc.set_missing_host_key_policy(paramiko.AutoAddPolicy())
b = cfg["targets"]["geegoo-bot"]["ssh"]
bc.connect(hostname=b["host"], username=b["user"], password=b.get("password"), timeout=30)
remote = '''
import json
from pymongo import MongoClient
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=", 1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=", 1)[1]
db = MongoClient(mongo_uri)[dbn]
users = list(db["user"].find({"mcp.token": {"$exists": True, "$ne": ""}}, {"email": 1, "mcp.token": 1}).limit(5))
print(json.dumps([{"id": str(u["_id"]), "email": u.get("email"), "token_prefix": (u.get("mcp") or {}).get("token", "")[:20]} for u in users], ensure_ascii=False))
'''
_, o, _ = bc.exec_command(f"python3 <<'PY'\n{remote}\nPY", timeout=60)
print(o.read().decode())
bc.close()
