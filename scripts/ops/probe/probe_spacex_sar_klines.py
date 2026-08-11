#!/usr/bin/env python3
"""Fetch SPCX 60m klines and compute SAR edge vs state on signal host."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r'''
import json, os, urllib.request

os.chdir("/root/apps/GeeGooSignal")
env_files = [".env", "/root/apps/GeeGooSignal/.env", "/root/apps/GeeGooData/.env"]
for ef in env_files:
    if not os.path.isfile(ef):
        continue
    for line in open(ef):
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            k, v = line.split("=", 1)
            os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

data_url = os.environ.get("GEEGOO_DATA_HTTP_URL") or os.environ.get("GEEGOODATA_URL") or "http://47.80.14.120:3300"
# keys
keys = []
for k in ("GEEGOO_DATA_SERVICE_TOKEN", "GEEGOO_DATA_API_KEY", "GEEGOODATA_API_KEY", "API_KEY", "GEEGOO_SIGNAL_DATA_API_KEY"):
    if os.environ.get(k):
        keys.append(os.environ[k])
print("data_url", data_url)
print("token keys tried from", [k for k in ("GEEGOO_DATA_SERVICE_TOKEN", "GEEGOO_DATA_API_KEY") if os.environ.get(k)])

body = {"code": "SPCX.US", "frequency": "60m", "limit": 120}
headers_base = {"Content-Type": "application/json"}
bars = None
for key in keys + [""]:
    for path in ["/v1/market/klines", "/getKlines", "/klines"]:
        h = dict(headers_base)
        if key:
            h["Authorization"] = "Bearer " + key
            h["X-API-Key"] = key
        url = data_url.rstrip("/") + path
        try:
            req = urllib.request.Request(url, data=json.dumps(body).encode(), headers=h, method="POST")
            with urllib.request.urlopen(req, timeout=45) as r:
                kl = json.loads(r.read().decode())
            print("OK", path, "key?", bool(key), "type", type(kl).__name__)
            if isinstance(kl, dict):
                bars = kl.get("data") or kl.get("klines") or kl.get("bars") or kl.get("list")
                if bars is None and "close" in str(kl)[:200]:
                    print("keys", list(kl.keys())[:20])
            elif isinstance(kl, list):
                bars = kl
            if bars:
                break
        except Exception as e:
            print("FAIL", path, "key?", bool(key), e)
    if bars:
        break

if not bars:
    raise SystemExit("no bars")

rows = []
for b in bars:
    rows.append({
        "t": b.get("time") or b.get("trade_date") or b.get("datetime"),
        "h": float(b.get("high") or 0),
        "l": float(b.get("low") or 0),
        "c": float(b.get("close") or 0),
    })
print("bars", len(rows), "last", rows[-1])

# Use same SAR as GeeGooSignal indicator package if possible via calling local API
# Local getBotSignal already shown continuous; compute edge with approximate SAR
accel, maximum = 0.02, 0.2
n = len(rows)
sar = [0.0] * n
bull = True
sar[0] = rows[0]["l"]
ep = rows[0]["h"]
af = accel
for i in range(1, n):
    prev = sar[i - 1]
    h, l = rows[i]["h"], rows[i]["l"]
    sar[i] = prev + af * (ep - prev)
    if bull:
        sar[i] = min(sar[i], rows[i - 1]["l"])
        if i >= 2:
            sar[i] = min(sar[i], rows[i - 2]["l"])
        if l < sar[i]:
            bull = False
            sar[i] = ep
            ep = l
            af = accel
        else:
            if h > ep:
                ep = h
                af = min(af + accel, maximum)
    else:
        sar[i] = max(sar[i], rows[i - 1]["h"])
        if i >= 2:
            sar[i] = max(sar[i], rows[i - 2]["h"])
        if h > sar[i]:
            bull = True
            sar[i] = ep
            ep = h
            af = accel
        else:
            if l < ep:
                ep = l
                af = min(af + accel, maximum)

flips = []
for i in range(1, n):
    prev = "SHORT" if sar[i - 1] > rows[i - 1]["c"] else "LONG"
    cur = "SHORT" if sar[i] > rows[i]["c"] else "LONG"
    if prev == "SHORT" and cur == "LONG":
        flips.append((rows[i]["t"], "BUY", round(rows[i]["c"], 4), round(sar[i], 4)))
    elif prev == "LONG" and cur == "SHORT":
        flips.append((rows[i]["t"], "SELL", round(rows[i]["c"], 4), round(sar[i], 4)))

cur = "SHORT" if sar[-1] > rows[-1]["c"] else "LONG"
prev = "SHORT" if sar[-2] > rows[-2]["c"] else "LONG"
edge = "HOLD"
if prev == "SHORT" and cur == "LONG":
    edge = "BUY"
elif prev == "LONG" and cur == "SHORT":
    edge = "SELL"
print("FLAG/state now:", cur, "close", rows[-1]["c"], "sar", round(sar[-1], 4))
print("SIGNAL/edge last bar:", edge)
print("recent flips (last 10):")
for f in flips[-10:]:
    print(" ", f)
'''


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    # GeeGooSignal host
    t = cfg["targets"].get("geegoo-signal") or cfg["targets"].get("geegoo-tradingsignal")
    s = t["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/probe_spacex_sar_klines.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=180)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
