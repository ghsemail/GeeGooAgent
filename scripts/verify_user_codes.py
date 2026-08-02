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
ids = [str(d["_id"]) for d in db.signal_index_db.find({"index.type":"flag"}).limit(5)]
key = os.environ["GEEGOO_SIGNAL_SIGNAL_API_KEY"]
data_url = os.environ["GEEGOO_DATA_HTTP_URL"].rstrip("/") + "/v1/market/klines"
tok = os.environ.get("GEEGOO_DATA_SERVICE_TOKEN","")

def klines(code):
    body = json.dumps({"code":code,"frequency":"daily","limit":100}).encode()
    req = urllib.request.Request(data_url, data=body, method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+tok})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return len(json.loads(r.read().decode()).get("bars") or [])
    except Exception as e:
        return -1

def flag(codes):
    body = {"code_list":codes,"type":"flag","frequency":"daily","signal_index_list":ids,"language":"cn"}
    req = urllib.request.Request("http://127.0.0.1:3200/getCodeListFlag", data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

codes = ["07552.HK","00700.HK","00772.HK","000858.SZ","002466.SZ","AAPL.US"]
print("index_ids", len(ids))
for code in codes:
    print(f"{code:12} klines={klines(code)}")
print("--- getCodeListFlag ---")
for row in flag(codes):
    print(row)
PY'"""
_, o, e = c.exec_command(cmd, timeout=120)
print((o.read()+e.read()).decode("utf-8", "replace"))
