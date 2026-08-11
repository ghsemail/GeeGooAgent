#!/usr/bin/env python3
"""List Mongo signal_index_db entries used by trading_operation."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r'''
import json, os, urllib.request

os.chdir("/root/apps/GeeGooSignal")
for line in open(".env"):
    line=line.strip()
    if line and not line.startswith("#") and "=" in line:
        k,v=line.split("=",1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

# catalog api
BASE=os.environ.get("GEEGOO_SIGNAL_CATALOG_API_URL","http://127.0.0.1:3210")
# if remote host in env, prefer local
BASE="http://127.0.0.1:3210"
KEY=os.environ.get("GEEGOO_SIGNAL_CATALOG_API_KEY","")

def post(path, body):
    req=urllib.request.Request(BASE+path, data=json.dumps(body).encode(),
        headers={"Content-Type":"application/json","Authorization":"Bearer "+KEY,"X-API-Key":KEY}, method="POST")
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

try:
    data=post("/getIndexSignal", {})
except Exception as e:
    print("getIndexSignal err", e)
    data=None

rows=[]
if isinstance(data, list):
    rows=data
elif isinstance(data, dict):
    rows=data.get("data") or data.get("list") or data.get("signal") or []
print("catalog_count", len(rows) if isinstance(rows, list) else type(rows))
sig=[]; flag=[]; other=[]
for item in rows if isinstance(rows, list) else []:
    idx=(item.get("index") or {})
    if not isinstance(idx, dict):
        idx={}
    name=item.get("name")
    if isinstance(name, dict):
        name=name.get("cn") or name.get("en") or str(name)
    entry={
        "name": name,
        "index": idx.get("index") or idx.get("name"),
        "type": idx.get("type"),
        "param": idx.get("param"),
        "frequency": item.get("frequency"),
    }
    t=(entry["type"] or "").lower()
    if t=="signal":
        sig.append(entry)
    elif t=="flag":
        flag.append(entry)
    else:
        other.append(entry)
print("type=signal", len(sig))
for e in sig:
    print(json.dumps(e, ensure_ascii=False))
print("type=flag", len(flag))
for e in flag:
    print(json.dumps({"name":e["name"],"index":e["index"],"type":e["type"]}, ensure_ascii=False))
print("other", len(other))
'''


def main():
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/list_signal_catalog.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=120)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()


if __name__ == "__main__":
    main()
