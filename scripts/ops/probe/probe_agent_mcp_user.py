#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))

ac = paramiko.SSHClient()
ac.set_missing_host_key_policy(paramiko.AutoAddPolicy())
a = cfg["targets"]["geegoo-agent"]["ssh"]
ac.connect(hostname=a["host"], username=a["user"], password=a.get("password"), timeout=30)
_, o, _ = ac.exec_command(
    "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); print(json.dumps({k:c.get(k) for k in ['mcp_token','api_key','user_id'] if c.get(k)}, ensure_ascii=False))\"",
    timeout=20,
)
agent_cfg = o.read().decode().strip()
ac.close()
print("agent_cfg", agent_cfg)

bc = paramiko.SSHClient()
bc.set_missing_host_key_policy(paramiko.AutoAddPolicy())
b = cfg["targets"]["geegoo-bot"]["ssh"]
bc.connect(hostname=b["host"], username=b["user"], password=b.get("password"), timeout=30)
remote = f'''
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
agent = json.loads({json.dumps(agent_cfg)})
token = agent.get("mcp_token", "")
users = list(db["user"].find({{"mcp.token": token}}, {{"_id": 1, "email": 1}}).limit(3))
if not users and token:
    users = list(db["user"].find({{"mcp.token": {{"$regex": token[:12]}}}}, {{"_id": 1, "email": 1, "mcp.token": 1}}).limit(3))
print(json.dumps({{"matched_users": [{{"id": str(u["_id"]), "email": u.get("email"), "token": (u.get("mcp") or {{}}).get("token", "")[:20]}} for u in users]}}, ensure_ascii=False))
'''
_, o, _ = bc.exec_command(f"python3 <<'PY'\n{remote}\nPY", timeout=60)
print(o.read().decode())
bc.close()
