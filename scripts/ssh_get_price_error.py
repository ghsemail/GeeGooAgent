#!/usr/bin/env python3
"""Fetch getCurrentPrice error details from Bot server."""

from __future__ import annotations

import os
import sys

import paramiko

PASS = os.environ.get("SSH_PASS", "")
HOST = "118.195.135.97"


def run(client: paramiko.SSHClient, cmd: str) -> str:
    print(f"--- {cmd[:90]}")
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=45)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    if out.strip():
        print(out[:4000])
    if err.strip():
        print("STDERR:", err[:800])
    return out


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username="ubuntu", password=PASS, timeout=25)

    run(
        client,
        r"""python3 - <<'PY'
import requests, traceback
url = 'http://127.0.0.1:5600/getCurrentPrice'
try:
    r = requests.post(url, json={'code': '00700.HK'}, timeout=30)
    print('HTTP', r.status_code)
    print('BODY', r.text[:800])
except Exception:
    traceback.print_exc()
PY""",
    )
    run(
        client,
        "grep -a 'getCurrentPrice\\|getPrice\\|Traceback\\|Error' /home/ubuntu/apps/TradingBot/api.out 2>/dev/null | tail -25",
    )
    run(
        client,
        r"""python3 - <<'PY'
import requests
# MCP layer
r = requests.post(
    'http://127.0.0.1:5700/getCurrentPrice',
    json={'code': '00700.HK'},
    headers={'Authorization': 'Bearer REPLACE'},
    timeout=30,
)
print('MCP without key:', r.status_code, r.text[:200])
PY""",
    )
    run(
        client,
        "grep -a 'get_current_price\\|getCurrentPrice' /home/ubuntu/apps/TradingBot/mcpapi.out 2>/dev/null | tail -10",
    )
    run(
        client,
        "sed -n '2188,2225p' /home/ubuntu/apps/TradingBot/mcpAPIServer.py",
    )
    run(
        client,
        "sed -n '3090,3115p' /home/ubuntu/apps/TradingBot/botAPIServer.py",
    )
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
