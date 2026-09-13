#!/usr/bin/env python3
"""Compare probe vs diagnose: how many bars are returned."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r"""
import json, os, urllib.request

os.chdir("/root/apps/GeeGooSignal")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

KEY = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY", "")
BASE = "http://127.0.0.1:3200"
body = {
    "code": "AAPL.US",
    "frequency": "daily",
    "months_back": 3,
    "side": "buy",
    "buy_signal": [{
        "index": "RSIThrehold",
        "type": "signal",
        "param": {"period": 25, "threholdBuy": 30, "threholdSell": 70},
    }],
}
headers = {
    "Content-Type": "application/json",
    "Authorization": "Bearer " + KEY,
    "X-API-Key": KEY,
}

def post(path):
    req = urllib.request.Request(
        BASE + path, data=json.dumps(body).encode(), headers=headers, method="POST"
    )
    with urllib.request.urlopen(req, timeout=90) as r:
        return json.loads(r.read().decode())

probe = post("/probeBotSignalSeries")
diag = post("/diagnoseBotSignalSeries")

print("=== probeBotSignalSeries ===")
print("top keys:", list(probe.keys()))
print("bars count:", len(probe.get("bars") or []))
if probe.get("bars"):
    print("first bar:", probe["bars"][0])
    print("last bar:", probe["bars"][-1])
br = probe.get("buy_rules") or []
if br:
    r0 = br[0]
    print("buy_rules[0] keys:", list(r0.keys()))
    vs = r0.get("value_series") or {}
    for k, v in vs.items():
        print(f"  value_series[{k}]: len={len(v)}, head={v[:3]}, tail={v[-3:]}")
    sig = r0.get("signal_series") or []
    print(f"  signal_series: len={len(sig)}, hits={sum(1 for x in sig if x==1)}")
    reasons = r0.get("reasons") or []
    print(f"  reasons: len={len(reasons)}, tail3={reasons[-3:]}")

print()
print("=== diagnoseBotSignalSeries ===")
print("top keys:", list(diag.keys()))
print("has bars field:", "bars" in diag)
print("range:", diag.get("range"))
dr = (diag.get("buy_rules") or [{}])[0]
print("buy_rules[0] keys:", list(dr.keys()))
print("has value_series:", "value_series" in dr)
print("has signal_series:", "signal_series" in dr)
print("sample_reasons count:", len(dr.get("sample_reasons") or []))
print("sample_reasons:", json.dumps(dr.get("sample_reasons"), ensure_ascii=False))
print("stats:", json.dumps(dr.get("stats"), ensure_ascii=False))
print("last_bar:", json.dumps(dr.get("last_bar"), ensure_ascii=False))
print("near_miss:", json.dumps(dr.get("near_miss"), ensure_ascii=False))
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/compare_probe_diagnose.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=120)
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
