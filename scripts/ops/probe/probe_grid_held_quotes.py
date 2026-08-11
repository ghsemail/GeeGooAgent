#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r'''
import os, json, urllib.request

os.chdir("/home/ubuntu/apps/GeeGooBot")
for line in open(".env"):
    line = line.strip()
    if line and not line.startswith("#") and "=" in line:
        k, v = line.split("=", 1)
        os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))

base = (os.environ.get("GEEGOO_DATA_HTTP_URL") or "").rstrip("/")
token = (
    os.environ.get("GEEGOO_DATA_SERVICE_TOKEN")
    or os.environ.get("GEEGOO_BOT_DATA_TOKEN")
    or ""
)
out = {"base": base, "quotes": {}}
for code in ["00700.HK", "01810.HK", "01818.HK"]:
    u = base + "/v1/market/quote"
    data = json.dumps({"code": code}).encode()
    req = urllib.request.Request(
        u,
        data=data,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {token}",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            body = json.loads(r.read().decode())
            # keep compact
            if isinstance(body, dict):
                d = body.get("data") if isinstance(body.get("data"), dict) else body
                out["quotes"][code] = {
                    k: d.get(k)
                    for k in (
                        "code", "name", "price", "last", "last_price", "current",
                        "close", "prev_close", "change", "change_pct",
                    )
                    if isinstance(d, dict) and k in d
                } or d
            else:
                out["quotes"][code] = body
    except Exception as e:
        out["quotes"][code] = {"error": str(e)}
print(json.dumps(out, ensure_ascii=False, indent=2, default=str))
'''


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    try:
        _, stdout, stderr = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=60)
        print(stdout.read().decode("utf-8", errors="replace"))
        err = stderr.read().decode("utf-8", errors="replace")
        if err.strip():
            print("STDERR:", err)
    finally:
        c.close()


if __name__ == "__main__":
    main()
