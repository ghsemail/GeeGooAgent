#!/usr/bin/env python3
import json, time, urllib.request
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
sig = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(sig["host"], username=sig["user"], password=sig.get("password"))

for cmd in [
    "cd /root/apps/GeeGooSignal && git fetch origin main && git reset --hard origin/main && git log -1 --oneline",
    "cd /root/apps/GeeGooSignal && printf '3\\n' | bash start.sh 2>&1 | tail -5",
]:
    print(">>>", cmd)
    _, o, e = c.exec_command(cmd, timeout=180)
    print((o.read()+e.read()).decode())

script = r"""bash -lc 'cd /root/apps/GeeGooSignal && set -a && source .env && set +a && python3 <<PY
import json, os, time, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_SIGNAL_MONGO_URI"])[os.environ.get("GEEGOO_SIGNAL_MONGO_DB","Signal_DB")]
codes = [d["code"] for d in db.stock_db.find().limit(9)]
if len(codes) < 3:
    codes = ["00700.HK","AAPL.US","07552.HK","00772.HK","01810.HK","600519.SH","000858.SZ","002466.SZ","TSLA.US"]
idx = [str(d["_id"]) for d in db.signal_index_db.find({"index.type":"flag"}).limit(5)]
key = os.environ["GEEGOO_SIGNAL_SIGNAL_API_KEY"]

def post_ms(body):
    req = urllib.request.Request("http://127.0.0.1:3200/getCodeListFlag", data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    t0 = time.perf_counter()
    with urllib.request.urlopen(req, timeout=120) as r:
        json.loads(r.read().decode())
    return round((time.perf_counter()-t0)*1000, 1)

body = {"code_list": codes, "type":"flag", "frequency":"5m", "signal_index_list": idx, "language":"cn"}
print("stocks", len(codes))
for i in range(2):
    print(f"run{i+1}_ms", post_ms(body))
PY'"""
print(">>> benchmark")
_, o, e = c.exec_command(script, timeout=180)
print((o.read()+e.read()).decode())
