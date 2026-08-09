#!/usr/bin/env python3
"""Deep probe: klines code formats + signal API with real index ids."""
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
sig_url = "http://127.0.0.1:3200"
sig_key = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY","")

def klines(code, freq="daily"):
    body = json.dumps({"code":code,"frequency":freq,"limit":100}).encode()
    req = urllib.request.Request(url, data=body, method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+tok})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            d = json.loads(r.read().decode())
            bars = d.get("bars") or []
            return len(bars), None
    except Exception as e:
        err = e.read().decode() if hasattr(e,"read") else str(e)
        return 0, err[:300]

def sig_post(path, body):
    data = json.dumps(body).encode()
    req = urllib.request.Request(sig_url+path, data=data, method="POST", headers={"Content-Type":"application/json","Authorization":"Bearer "+sig_key})
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

print("=== Klines formats ===")
for code in ("0700.HK","00700.HK","HK.00700","07552.HK","000858.SZ","600519.SH","AAPL.US"):
    n, err = klines(code)
    print(f"{code:12} bars={n}" + (f" err={err}" if err else ""))

idx = "6623e226f71be5ed2500ecfa"
print("\n=== getDashboardSignal 00700.HK ===")
for code in ("0700.HK","00700.HK"):
    try:
        d = sig_post("/getDashboardSignal", {"code":code,"frequency":"daily","signal_index_list":[idx]})
        print(code, "total=", d.get("total"), "signals=", len(d.get("signal") or []))
    except Exception as e:
        print(code, "ERR", (e.read().decode() if hasattr(e,"read") else str(e))[:200])

print("\n=== getCodeListFlag ===")
try:
    d = sig_post("/getCodeListFlag", {"code_list":["00700.HK","07552.HK","AAPL.US"],"signal_index_list":[idx]})
    for row in (d.get("code_list") or d.get("data") or d)[:5]:
        if isinstance(row, dict):
            print(row)
except Exception as e:
    print("ERR", (e.read().decode() if hasattr(e,"read") else str(e))[:300])

print("\nENV PROVIDER", os.environ.get("GEEGOO_DATA_PROVIDER"), "LOCAL", os.environ.get("GEEGOO_DATA_MARKET_LOCAL"))
PY'"""
_, o, e = c.exec_command(cmd, timeout=120)
print((o.read()+e.read()).decode("utf-8", "replace"))
