#!/usr/bin/env python3
"""Compare SpaceX SAR signal (edge) vs flag (state) vs live getBotSignal."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r'''
import json, os, urllib.request
from bson import ObjectId
from pymongo import MongoClient

os.chdir("/home/ubuntu/apps/GeeGooBot")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

db = MongoClient(os.environ["GEEGOO_BOT_MONGO_URI"])[os.environ["GEEGOO_BOT_MONGO_DB"]]
bot = db["dca_bot"].find_one({"code": "SPCX.US"})
sig = (bot or {}).get("signal") or {}
print("bot signal config:")
print(json.dumps(sig, ensure_ascii=False, indent=2))

base = os.environ["GEEGOO_SIGNAL_SIGNAL_API_URL"]
key = os.environ["GEEGOO_SIGNAL_SIGNAL_API_KEY"]
data_url = os.environ.get("GEEGOO_DATA_HTTP_URL", "http://47.80.14.120:3300")

def post(url, body, headers=None):
    h = {"Content-Type": "application/json"}
    if headers:
        h.update(headers)
    req = urllib.request.Request(url, data=json.dumps(body).encode(), headers=h, method="POST")
    with urllib.request.urlopen(req, timeout=60) as r:
        return json.loads(r.read().decode())

# 1) live getBotSignal full config
full = {
    "code": "SPCX.US",
    "frequency": "60m",
    "buy_signal": sig.get("buy_signal"),
    "sell_signal": sig.get("sell_signal"),
}
print("\n=== live getBotSignal (full bot config) ===")
print(json.dumps(post(base + "/getBotSignal", full, {"Authorization": "Bearer " + key, "X-API-Key": key}), ensure_ascii=False)[:1500])

# 2) SAR alone as signal
sar_only = {
    "code": "SPCX.US",
    "frequency": "60m",
    "buy_signal": [{"index": "SAR", "param": {"acceleration": "0.02", "maximum": "0.2"}, "type": "signal"}],
    "sell_signal": [{"index": "nosignal", "param": {}, "type": ""}],
}
print("\n=== getBotSignal SAR type=signal ONLY ===")
print(json.dumps(post(base + "/getBotSignal", sar_only, {"Authorization": "Bearer " + key, "X-API-Key": key}), ensure_ascii=False)[:1200])

# 3) SAR alone as flag
sar_flag = {
    "code": "SPCX.US",
    "frequency": "60m",
    "buy_signal": [{"index": "SAR", "param": {"acceleration": "0.02", "maximum": "0.2"}, "type": "flag"}],
    "sell_signal": [{"index": "nosignal", "param": {}, "type": ""}],
}
print("\n=== getBotSignal SAR type=flag ONLY ===")
print(json.dumps(post(base + "/getBotSignal", sar_flag, {"Authorization": "Bearer " + key, "X-API-Key": key}), ensure_ascii=False)[:1200])

# 4) fetch klines and compute edge vs state locally with a simple SAR
# Use GeeGooData klines
print("\n=== klines + local SAR edge/state ===")
kl_body = {"code": "SPCX.US", "frequency": "60m", "limit": 100}
# try common paths
bars = None
for path in ["/getKlines", "/klines", "/v1/klines", "/market/klines"]:
    try:
        kl = post(data_url + path, kl_body)
        print("klines via", path, "keys", list(kl.keys())[:10] if isinstance(kl, dict) else type(kl))
        if isinstance(kl, dict):
            bars = kl.get("data") or kl.get("klines") or kl.get("bars") or kl.get("list")
        if bars:
            break
    except Exception as e:
        print("fail", path, e)

if not bars:
    # try signal host analyze or data via tradingbot env
    print("no bars from data url; try geegoodata via signal market internally skipped")
else:
    # normalize
    rows = []
    for b in bars:
        if isinstance(b, dict):
            rows.append({
                "t": b.get("time") or b.get("trade_date") or b.get("datetime"),
                "h": float(b.get("high") or b.get("High") or 0),
                "l": float(b.get("low") or b.get("Low") or 0),
                "c": float(b.get("close") or b.get("Close") or 0),
            })
    print("bars", len(rows), "first", rows[0] if rows else None, "last", rows[-1] if rows else None)

    # parabolic SAR (same spirit as talib / Go)
    accel, maximum = 0.02, 0.2
    n = len(rows)
    sar = [0.0]*n
    if n >= 2:
        bull = rows[1]["c"] >= rows[0]["c"]
        sar[0] = rows[0]["l"] if bull else rows[0]["h"]
        ep = rows[0]["h"] if bull else rows[0]["l"]
        af = accel
        for i in range(1, n):
            prev = sar[i-1]
            h, l, c = rows[i]["h"], rows[i]["l"], rows[i]["c"]
            sar[i] = prev + af * (ep - prev)
            if bull:
                sar[i] = min(sar[i], rows[i-1]["l"])
                if i >= 2:
                    sar[i] = min(sar[i], rows[i-2]["l"])
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
                sar[i] = max(sar[i], rows[i-1]["h"])
                if i >= 2:
                    sar[i] = max(sar[i], rows[i-2]["h"])
                if h > sar[i]:
                    bull = True
                    sar[i] = ep
                    ep = h
                    af = accel
                else:
                    if l < ep:
                        ep = l
                        af = min(af + accel, maximum)

    # state (flag) and edge (signal)
    flips = []
    last_state = None
    for i in range(n):
        state = "LONG" if rows[i]["c"] >= sar[i] else "SHORT"  # close above SAR => long (matches Go: sar>close => short)
        # Go: if sar[i] > closes[i] => Short else Long
        state = "SHORT" if sar[i] > rows[i]["c"] else "LONG"
        edge = "HOLD"
        if i > 0:
            prev = "SHORT" if sar[i-1] > rows[i-1]["c"] else "LONG"
            if prev == "SHORT" and state == "LONG":
                edge = "BUY"
                flips.append((rows[i]["t"], "BUY", rows[i]["c"], round(sar[i],4)))
            elif prev == "LONG" and state == "SHORT":
                edge = "SELL"
                flips.append((rows[i]["t"], "SELL", rows[i]["c"], round(sar[i],4)))
        last_state = state
    print("current FLAG/state:", last_state, "close", rows[-1]["c"], "sar", round(sar[-1],4), "time", rows[-1]["t"])
    print("current SIGNAL/edge on last bar:", end=" ")
    if n > 1:
        prev = "SHORT" if sar[-2] > rows[-2]["c"] else "LONG"
        cur = "SHORT" if sar[-1] > rows[-1]["c"] else "LONG"
        if prev == "SHORT" and cur == "LONG":
            print("BUY")
        elif prev == "LONG" and cur == "SHORT":
            print("SELL")
        else:
            print("HOLD (no flip this bar)")
    print("last 8 flips:")
    for f in flips[-8:]:
        print(" ", f)
'''


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/probe_spacex_sar.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=180)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
