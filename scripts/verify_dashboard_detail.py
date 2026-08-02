#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))

cmd = r"""bash -lc 'cd /root/apps/GeeGooSignal && set -a && source .env && set +a && python3 <<PY
import json, os, urllib.request
from pymongo import MongoClient

db = MongoClient(os.environ["GEEGOO_SIGNAL_MONGO_URI"])[os.environ.get("GEEGOO_SIGNAL_MONGO_DB","Signal_DB")]
ids = [str(d["_id"]) for d in db.signal_index_db.find({"index.type":"flag"}).limit(10)]
key = os.environ["GEEGOO_SIGNAL_SIGNAL_API_KEY"]

def dashboard(code):
    body = {"code":code,"type":"flag","frequency":"daily","signal_index_list":ids,"language":"cn"}
    req = urllib.request.Request("http://127.0.0.1:3200/getDashboardSignal", data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

for code in ("00700.HK","07552.HK","AAPL.US"):
    d = dashboard(code)
    sigs = d.get("signal") or []
    print("===", code, "total=", d.get("total"), "signal_len=", len(sigs))
    for s in sigs[:3]:
        print(" ", s.get("name"), "signal=", s.get("signal"), "type=", s.get("type"))
    if len(sigs) > 3:
        vals = [x.get("signal") for x in sigs]
        print("  all_signals", vals)
PY'"""
_, o, e = c.exec_command(cmd, timeout=120)
print((o.read()+e.read()).decode("utf-8", "replace"))
