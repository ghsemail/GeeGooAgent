#!/usr/bin/env python3
import json, time, urllib.request
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
bot = cfg["targets"]["geegoo-bot"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(bot["host"], username=bot["user"], password=bot.get("password"))

script = r"""bash -lc 'cd /home/ubuntu/apps/GeeGooBot && set -a && source .env && set +a && python3 <<PY
import json, os, time, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
doc = db.user_security.find_one()
uid = str(doc["user_id"])
codes = [d["code"] for d in db.user_security.find({"user_id": doc["user_id"]})]
app_key = os.environ["GEEGOO_BOT_APP_API_KEY"]
sig_key = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY", "")
idx = ["6623e226f71be5ed2500ecfa","662492aa585ef3df59f8bb8d","662c9459c4cee7ffb800d0a3"]

def post_ms(url, body, key=None):
    data = json.dumps(body).encode()
    headers = {"Content-Type":"application/json"}
    if key: headers["Authorization"] = "Bearer "+key
    t0 = time.perf_counter()
    req = urllib.request.Request(url, data=data, method="POST", headers=headers)
    with urllib.request.urlopen(req, timeout=120) as r:
        json.loads(r.read().decode())
    return round((time.perf_counter()-t0)*1000, 1)

print("stocks", len(codes))
for label, n in [("1_stock",1),("3_stocks",min(3,len(codes))),("all",len(codes))]:
    subset = codes[:n]
    ms = post_ms("http://146.56.225.252:3200/getCodeListFlag", {
        "code_list": subset, "type":"flag", "frequency":"5m", "signal_index_list": idx, "language":"cn"
    }, sig_key)
    print(f"signal_{label}_ms", ms, "per_stock", round(ms/max(n,1),1))

for _ in range(3):
    ms = post_ms("http://127.0.0.1:3100/getUserStockTrend", {
        "user_id": uid, "type":"flag", "frequency":"5m", "signal_index_list": idx, "language":"cn"
    }, app_key)
    print("app_getUserStockTrend_ms", ms)
PY'"""
_, o, e = c.exec_command(script, timeout=180)
print((o.read()+e.read()).decode("utf-8", "replace"))
