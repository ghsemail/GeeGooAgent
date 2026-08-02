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
for code in ("0700.HK","000858.SZ","AAPL.US"):
  for freq in ("daily","5m","60m"):
    body = json.dumps({"code":code,"frequency":freq,"limit":100}).encode()
    req = urllib.request.Request(url, data=body, method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+tok})
    try:
      with urllib.request.urlopen(req, timeout=30) as r:
        d = json.loads(r.read().decode())
        bars = d.get("bars") or []
        print(code, freq, "bars", len(bars))
    except Exception as e:
      print(code, freq, "ERR", (e.read().decode() if hasattr(e,"read") else str(e))[:200])
print("PROVIDER", os.environ.get("GEEGOO_DATA_PROVIDER"))
print("MARKET_LOCAL", os.environ.get("GEEGOO_DATA_MARKET_LOCAL"))
PY'"""
_, o, e = c.exec_command(cmd, timeout=90)
print((o.read()+e.read()).decode())
