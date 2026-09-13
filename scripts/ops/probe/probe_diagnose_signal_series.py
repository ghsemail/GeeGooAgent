#!/usr/bin/env python3
"""Smoke POST /diagnoseBotSignalSeries on GeeGooSignal signal-api."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r"""
import json, os, urllib.request, urllib.error

os.chdir("/root/apps/GeeGooSignal")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

BASE = "http://127.0.0.1:3200"
KEY = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY", "")
body = {
    "code": "AAPL.US",
    "frequency": "daily",
    "months_back": 3,
    "side": "buy",
    "buy_signal": [
        {
            "index": "RSIThrehold",
            "type": "signal",
            "param": {"period": 25, "threholdBuy": 30, "threholdSell": 70},
        }
    ],
}


def post(path, payload):
    req = urllib.request.Request(
        BASE + path,
        data=json.dumps(payload).encode(),
        headers={
            "Content-Type": "application/json",
            "Authorization": "Bearer " + KEY,
            "X-API-Key": KEY,
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=90) as r:
            return r.status, json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        raw = e.read().decode(errors="replace")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw[:800]}


status, data = post("/diagnoseBotSignalSeries", body)
print("HTTP", status)
if status != 200:
    print(json.dumps(data, ensure_ascii=False, indent=2)[:2000])
    raise SystemExit(1)

print("verdict", data.get("verdict"))
print("summary", data.get("summary"))
print("hits", data.get("hits"))
rules = data.get("buy_rules") or []
if rules:
    r0 = rules[0]
    print("rule", r0.get("index"), "param", r0.get("param"))
    print("algorithm", (r0.get("algorithm") or "")[:120])
    stats = r0.get("stats") or {}
    cols = stats.get("columns") or []
    if cols:
        print("stats", cols[0])
    lb = r0.get("last_bar") or {}
    print("last_bar", lb.get("reason"))
    nm = r0.get("near_miss") or {}
    if nm:
        print("near_miss", nm.get("note") or nm.get("reason"))
print("OK")
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/probe_diagnose_signal.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=120)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    print(out)
    c.close()
    return 0 if "OK" in out else 1


if __name__ == "__main__":
    raise SystemExit(main())
