#!/usr/bin/env python3
"""Sync Signal_DB admin mcp_token to GeeGooBot QT_DB via agent-api ops sync."""
from __future__ import annotations

import json
import re
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
SYNC_URL = "http://118.195.135.97:3110/op_agent/v1/ops/sync-mcp-token"

SIGNAL_ADMINS_PY = r"""
import json, subprocess, sys
names = subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines()
mongo = next((n for n in names if 'mongo' in n.lower()), '')
if not mongo:
    print('MONGO_ERR no mongo container')
    sys.exit(1)
cmd = [
    'docker', 'exec', mongo, 'mongosh', 'Signal_DB', '--quiet',
    '--eval', 'JSON.stringify(db.admin.find({},{username:1,mcp_token:1}).toArray())'
]
raw = subprocess.check_output(cmd, timeout=30).decode().strip()
print(raw or '[]')
"""


def ssh_run(target: str, cmd: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    _, o, e = c.exec_command(cmd, timeout=90)
    out = o.read().decode("utf-8", errors="replace").strip()
    err = e.read().decode("utf-8", errors="replace").strip()
    c.close()
    if err:
        print(f"[{target} stderr] {err}")
    return out


def load_bot_agent_key() -> str:
    out = ssh_run(
        "geegoo-bot",
        "grep -E '^GEEGOO_BOT_AGENT_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env | head -1",
    )
    if not out:
        raise SystemExit("GEEGOO_BOT_AGENT_API_KEY not found on bot host")
    return out.split("=", 1)[1].strip().strip('"')


def main() -> None:
    api_key = load_bot_agent_key()
    raw = ssh_run("geegoo-signal", f"python3 <<'PY'\n{SIGNAL_ADMINS_PY}\nPY")
    if not raw or raw.startswith("MONGO_ERR"):
        raise SystemExit(f"failed to read Signal_DB admins: {raw!r}")
    admins = json.loads(raw)
    print(f"found {len(admins)} admin(s) in Signal_DB")
    for doc in admins:
        token = (doc.get("mcp_token") or "").strip()
        user = (doc.get("username") or "").strip()
        if not user or not token:
            print(" skip", user or doc, "no token/username")
            continue
        body = json.dumps({"username": user, "mcp_token": token}).encode()
        req = urllib.request.Request(
            SYNC_URL,
            data=body,
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {api_key}",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=20) as resp:
                print(" synced", user, resp.status)
        except urllib.error.HTTPError as ex:
            detail = ex.read().decode("utf-8", errors="replace")[:200]
            print(" FAIL", user, ex.code, detail)


if __name__ == "__main__":
    main()
