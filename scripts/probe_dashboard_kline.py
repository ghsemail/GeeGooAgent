#!/usr/bin/env python3
"""Probe production getDashboardKline response."""
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
key = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY","")

def post(path, body):
    req = urllib.request.Request("http://127.0.0.1:3200/"+path, data=json.dumps(body).encode(), method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+key})
    with urllib.request.urlopen(req, timeout=120) as r:
        return json.loads(r.read().decode())

for code in ("AAPL.US","00700.HK","600519.SH","000858.SZ"):
    try:
        res = post("getDashboardKline", {"code": code})
        print("===", code, "type", type(res).__name__, "len", len(res) if isinstance(res,list) else None)
        if isinstance(res, list):
            for i, row in enumerate(res):
                sig = row.get("signal") or []
                total = row.get("total") or {}
                print("  [%d] freq=%s signals=%d total=%s" % (i, row.get("frequency"), len(sig), total))
                if sig:
                    print("    first_signal", sig[0])
        else:
            print(" ", res)
    except Exception as e:
        err = e.read().decode() if hasattr(e,"read") else str(e)
        print(code, "ERR", err[:300])
PY'"""
_, o, e = c.exec_command(cmd, timeout=180)
print((o.read()+e.read()).decode("utf-8", "replace"))
