#!/usr/bin/env python3
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    # read signal token
    sc = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(sc["host"], username=sc["user"], password=sc.get("password"), timeout=60)
    py = r"""
import json, subprocess
names = subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines()
mongo = next((n for n in names if 'mongo' in n.lower()), '')
raw = subprocess.check_output(['docker','exec',mongo,'mongosh','Signal_DB','--quiet','--eval',
 "JSON.stringify(db.admin.find({username:'ghsemail'},{mcp_token:1}).toArray())"], text=True).strip()
print(raw)
"""
    _, o, _ = c.exec_command(f"python3 <<'PY'\n{py}\nPY", timeout=60)
    sig = json.loads(o.read().decode().strip() or "[]")
    c.close()
    token = (sig[0].get("mcp_token") if sig else "") or ""
    print("signal token prefix:", token[:16])

    # read bot QT_DB admin token
    bc = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(bc["host"], username=bc["user"], password=bc.get("password"), timeout=60)
    cmd = (
        "docker ps --format '{{.Names}}' | grep -i mongo | head -1 | "
        "xargs -I{} docker exec {} mongosh QT_DB --quiet --eval "
        "\"JSON.stringify(db.admin.find({username:'ghsemail'},{mcp_token:1,username:1}).toArray())\""
    )
    _, o, e = c.exec_command(cmd, timeout=60)
    bot_raw = o.read().decode().strip()
    print("bot admin:", bot_raw[:200])
    c.close()

    url = "http://146.56.225.252:8088/op_agent/v1/dashboard/data"
    headers = {
        "Authorization": f"Bearer {KEY}",
        "X-MCP-Token": token,
        "X-Client-Source": "trading_operation",
    }
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            print("dashboard:", r.status, "OK")
    except urllib.error.HTTPError as ex:
        print("dashboard FAIL:", ex.code, ex.read()[:120])


if __name__ == "__main__":
    main()
