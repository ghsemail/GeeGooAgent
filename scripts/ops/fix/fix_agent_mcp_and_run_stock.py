#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

USER = "6366170502d5c175fd586fe8"
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))

bc = paramiko.SSHClient()
bc.set_missing_host_key_policy(paramiko.AutoAddPolicy())
b = cfg["targets"]["geegoo-bot"]["ssh"]
bc.connect(hostname=b["host"], username=b["user"], password=b.get("password"), timeout=30)
remote = f'''
import json
from bson import ObjectId
from pymongo import MongoClient
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=", 1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=", 1)[1]
db = MongoClient(mongo_uri)[dbn]
user = db["user"].find_one({{"_id": ObjectId("{USER}")}})
token = ((user or {{}}).get("mcp") or {{}}).get("token", "")
print(json.dumps({{"user": "{USER}", "mcp_token_len": len(token), "mcp_token": token}}))
'''
_, o, _ = bc.exec_command(f"python3 <<'PY'\n{remote}\nPY", timeout=60)
user_data = json.loads(o.read().decode().strip())
bc.close()
print("user_mcp_len", user_data.get("mcp_token_len"))

ac = paramiko.SSHClient()
ac.set_missing_host_key_policy(paramiko.AutoAddPolicy())
a = cfg["targets"]["geegoo-agent"]["ssh"]
ac.connect(hostname=a["host"], username=a["user"], password=a.get("password"), timeout=30)
remote2 = f'''
import json, os, subprocess
cfg_path = os.path.expanduser("~/.geegoo/config.json")
cfg = json.load(open(cfg_path, encoding="utf-8"))
cfg["mcp_token"] = {json.dumps(user_data.get("mcp_token", ""))}
json.dump(cfg, open(cfg_path, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
run = subprocess.run([
    os.path.expanduser("~/.geegoo/bin/geegoo"), "run", "--config", cfg_path,
    "--market", "CN", "--report-date", "2026-08-07", "premarket_stock",
], capture_output=True, text=True, timeout=900)
print(json.dumps({{
    "exit": run.returncode,
    "stderr_tail": (run.stderr or "")[-1500:],
    "stdout_tail": (run.stdout or "")[-1500:],
}}, ensure_ascii=False))
'''
_, o, _ = ac.exec_command(f"python3 <<'PY'\n{remote2}\nPY", timeout=920)
print(o.read().decode())
ac.close()
