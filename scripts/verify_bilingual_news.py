#!/usr/bin/env python3
import json
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r'''
import json, urllib.request
body = json.dumps({"stock_list": [{"code": "TSLA.US", "name": {"init": "TSLA"}}], "language": "cn"}).encode()
req = urllib.request.Request("http://127.0.0.1:3300/getStockNews", data=body,
    headers={"Content-Type": "application/json"}, method="POST")
items = json.loads(urllib.request.urlopen(req, timeout=30).read())
print("count", len(items))
if items:
    t = items[0].get("title", {})
    print("keys", sorted(t.keys()))
    print("cn", (t.get("cn") or "")[:80])
    print("en", (t.get("en") or "")[:80])
'''


def check(target: str) -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
    _, o, e = c.exec_command(f"python3 - <<'PY'\n{REMOTE}\nPY", timeout=30)
    print(f"=== {target} ({ssh['host']}) ===")
    print((o.read() + e.read()).decode("utf-8", errors="replace"))
    c.close()


if __name__ == "__main__":
    check("geegoo-data")
