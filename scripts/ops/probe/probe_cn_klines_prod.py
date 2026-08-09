#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-data"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))

script = r"""bash -lc 'cd /root/apps/GeeGooData && set -a && source .env && set +a && python3 <<PY
import json, os, urllib.request
url="http://127.0.0.1:3300/v1/market/klines"
tok=os.environ.get("GEEGOO_DATA_SERVICE_TOKEN","")
for code,freq in [("600519.SH","daily"),("600519.SH","5m"),("000858.SZ","daily"),("000858.SZ","5m")]:
  body=json.dumps({"code":code,"frequency":freq,"limit":100}).encode()
  req=urllib.request.Request(url,data=body,method="POST",headers={"Content-Type":"application/json","Authorization":"Bearer "+tok})
  try:
    with urllib.request.urlopen(req,timeout=60) as r:
      d=json.loads(r.read().decode())
      bars=d.get("bars") or []
      print(code,freq,"bars",len(bars))
  except Exception as e:
    err=e.read().decode() if hasattr(e,"read") else str(e)
    print(code,freq,"ERR",err[:200])
PY'"""
_, o, e = c.exec_command(script, timeout=180)
print((o.read() + e.read()).decode("utf-8", "replace"))
