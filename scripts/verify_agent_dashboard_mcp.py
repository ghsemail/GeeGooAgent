#!/usr/bin/env python3
"""Verify Agent dashboard with synced ops MCP token."""
from __future__ import annotations

import json
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
URL = "http://146.56.225.252:8088/op_agent/v1/dashboard/data"

SIGNAL_ADMINS = r"""
import json, subprocess, sys
names = subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines()
mongo = next((n for n in names if 'mongo' in n.lower()), '')
cmd = ['docker','exec',mongo,'mongosh','Signal_DB','--quiet','--eval',
       "JSON.stringify(db.admin.find({username:'ghsemail'},{mcp_token:1,username:1}).toArray())"]
print(subprocess.check_output(cmd, timeout=30).decode().strip())
"""


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    _, o, _ = c.exec_command(f"python3 <<'PY'\n{SIGNAL_ADMINS}\nPY", timeout=60)
    raw = o.read().decode().strip()
    c.close()
    admins = json.loads(raw or "[]")
    if not admins:
        print("no ghsemail admin")
        sys.exit(1)
    token = (admins[0].get("mcp_token") or "").strip()
    if not token:
        print("ghsemail has no mcp_token in Signal_DB")
        sys.exit(1)
    req = urllib.request.Request(
        URL,
        headers={"X-MCP-Token": token, "X-Client-Source": "verify"},
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            body = resp.read(200)
            print(f"OK {resp.status} {body[:160]!r}")
    except urllib.error.HTTPError as e:
        print(f"HTTP {e.code} {e.read()[:200]!r}")
        sys.exit(1)


if __name__ == "__main__":
    main()
