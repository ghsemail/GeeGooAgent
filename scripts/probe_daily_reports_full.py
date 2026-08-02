#!/usr/bin/env python3
import json
import urllib.request
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
USER = "6366170502d5c175fd586fe8"
BOT = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"

REMOTE = f'''
import json, urllib.request
from bson import ObjectId
from pymongo import MongoClient

USER = "{USER}"
BOT = "{BOT}"

def post(url, body, key=None):
    data = json.dumps(body).encode()
    h = {{"Content-Type": "application/json"}}
    if key:
        h["Authorization"] = f"Bearer {{key}}"
    req = urllib.request.Request(url, data=data, headers=h, method="POST")
    with urllib.request.urlopen(req, timeout=60) as r:
        return r.status, json.loads(r.read())

# full limit
c,d = post("http://127.0.0.1:3140/reports/daily", {{"user_id": USER, "limit_per_phase": 200}}, BOT)
data = d.get("data", {{}})
for phase in ["pre_market", "post_market", "intraday"]:
    rows = data.get(phase) or []
    dates = {{}}
    for r in rows:
        rd = str(r.get("report_date") or r.get("session_date") or "")[:10]
        dates[rd] = dates.get(rd, 0) + 1
    top_dates = sorted(dates.items(), reverse=True)[:5]
    print(f"{{phase}}: total={{len(rows)}} top_dates={{top_dates}}")

# agent mcp token from config
cfg = json.load(open("/home/ubuntu/.geegoo/config.json"))
mcp = cfg.get("mcp_token", "")
print("agent_mcp_token", bool(mcp))
if mcp:
    for path, body in [
        ("getReportBotCodes", {{"mcp_token": mcp}}),
        ("getPreMarketReports", {{"mcp_token": mcp, "code": "00700.HK", "period": "daily"}}),
    ]:
        try:
            c,r = post(f"http://127.0.0.1:3120/{{path}}", body, BOT)
            print(path, "http", c, "code", r.get("code"), "msg", r.get("message"))
            if path.endswith("getPreMarketReports") and r.get("data"):
                print("  reports", len(r.get("data") or []))
        except Exception as e:
            print(path, "ERR", e)

# user mcp in mongo
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=",1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=",1)[1]
user = MongoClient(mongo_uri)[dbn]["user"].find_one({{"_id": ObjectId(USER)}})
print("user_mcp_token", bool(((user or {{}}).get("mcp") or {{}}).get("token")))
'''

cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
_, o, e = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=90)
print((o.read() + e.read()).decode("utf-8", errors="replace"))
c.close()
