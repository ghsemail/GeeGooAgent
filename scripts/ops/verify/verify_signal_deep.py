#!/usr/bin/env python3
"""Deep probe GeeGooSignal indexquery path."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(cmd: str) -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=120)
    print((o.read() + e.read()).decode("utf-8", "replace"))
    c.close()


script = r"""bash -lc 'cd /root/apps/GeeGooSignal && set -a && source .env && set +a && python3 <<'"'"'PY'"'"'
import json, os, urllib.request
from pymongo import MongoClient

uri = os.environ["GEEGOO_SIGNAL_MONGO_URI"]
dbn = os.environ.get("GEEGOO_SIGNAL_MONGO_DB", "Signal_DB")
data_url = os.environ.get("GEEGOO_DATA_HTTP_URL", "")
data_token = os.environ.get("GEEGOO_DATA_SERVICE_TOKEN", "")
key = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY", "")

db = MongoClient(uri)[dbn]
ids = [d["_id"] for d in db.signal_index_db.find({"index.type": "flag"}).limit(3)]
id_strs = [str(x) for x in ids]
print("index_docs:")
for d in db.signal_index_db.find({"_id": {"$in": ids}}):
    idx = d.get("index", {})
    name = d.get("name", {})
    print(" -", d["_id"], "index=", idx.get("index"), "type=", idx.get("type"), "name.cn=", name.get("cn") if isinstance(name, dict) else name)

# test data-api kline
code = "0700.HK"
for freq in ("daily", "5m", "60m"):
    url = data_url.rstrip("/") + "/getKline"
    body = {"code": code, "frequency": freq, "limit": 100}
    req = urllib.request.Request(url, data=json.dumps(body).encode(), method="POST")
    req.add_header("Content-Type", "application/json")
    if data_token:
        req.add_header("Authorization", f"Bearer {data_token}")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = json.loads(resp.read().decode())
            bars = raw if isinstance(raw, list) else raw.get("data") or raw.get("bars") or []
            print(f"DATA_API {freq} bars={len(bars)} sample_keys={list(bars[0].keys()) if bars else None}")
    except Exception as e:
        err = e.read().decode() if hasattr(e, "read") else str(e)
        print(f"DATA_API {freq} FAIL", err[:300])

# dashboard with real ids
body = {
    "code": "0700.HK",
    "type": "flag",
    "frequency": "daily",
    "signal_index_list": id_strs,
    "language": "cn",
}
req = urllib.request.Request(
    "http://127.0.0.1:3200/getDashboardSignal",
    data=json.dumps(body).encode(),
    method="POST",
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {key}"},
)
with urllib.request.urlopen(req, timeout=60) as resp:
    out = json.loads(resp.read().decode())
print("dashboard", json.dumps(out, ensure_ascii=False)[:1200])

# code list flag
body2 = {
    "code_list": ["0700.HK", "AAPL.US"],
    "type": "flag",
    "frequency": "daily",
    "signal_index_list": id_strs,
    "language": "cn",
}
req2 = urllib.request.Request(
    "http://127.0.0.1:3200/getCodeListFlag",
    data=json.dumps(body2).encode(),
    method="POST",
    headers={"Content-Type": "application/json", "Authorization": f"Bearer {key}"},
)
with urllib.request.urlopen(req2, timeout=60) as resp2:
    out2 = json.loads(resp2.read().decode())
print("codeListFlag", json.dumps(out2, ensure_ascii=False)[:800])
PY'"""

if __name__ == "__main__":
    run(script)
