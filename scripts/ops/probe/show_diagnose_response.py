#!/usr/bin/env python3
"""Fetch and print full /diagnoseBotSignalSeries JSON response."""
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

req = urllib.request.Request(
    BASE + "/diagnoseBotSignalSeries",
    data=json.dumps(body).encode(),
    headers={
        "Content-Type": "application/json",
        "Authorization": "Bearer " + KEY,
        "X-API-Key": KEY,
    },
    method="POST",
)
with urllib.request.urlopen(req, timeout=90) as r:
    data = json.loads(r.read().decode())
print(json.dumps(data, ensure_ascii=False, indent=2))
"""


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    p = "/tmp/show_diagnose_response.py"
    with c.open_sftp().file(p, "w") as f:
        f.write(REMOTE)
    _, o, e = c.exec_command(f"python3 {p}", timeout=120)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    print(out)
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
