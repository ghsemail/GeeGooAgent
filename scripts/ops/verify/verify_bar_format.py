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

url = os.environ["GEEGOO_DATA_HTTP_URL"].rstrip("/") + "/v1/market/klines"
tok = os.environ.get("GEEGOO_DATA_SERVICE_TOKEN","")

def sample(code):
    body = json.dumps({"code":code,"frequency":"daily","limit":3}).encode()
    req = urllib.request.Request(url, data=body, method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+tok})
    with urllib.request.urlopen(req, timeout=30) as r:
        d = json.loads(r.read().decode())
    bars = d.get("bars") or []
    print("===", code, "count", len(bars), "source", d.get("source"))
    if bars:
        print(" first bar", bars[0])
        print(" types", {k:type(v).__name__ for k,v in bars[0].items()})

for code in ("00700.HK","AAPL.US"):
    try:
        sample(code)
    except Exception as e:
        print(code, "ERR", (e.read().decode() if hasattr(e,"read") else str(e))[:300])
PY'"""
_, o, e = c.exec_command(cmd, timeout=60)
print((o.read()+e.read()).decode("utf-8", "replace"))
