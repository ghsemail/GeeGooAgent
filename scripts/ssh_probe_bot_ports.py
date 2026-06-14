#!/usr/bin/env python3
"""Probe Bot server local service ports."""

from __future__ import annotations

import os
import sys

import paramiko

PASS = os.environ.get("SSH_PASS", "")
HOST = "118.195.135.97"


def run(client: paramiko.SSHClient, cmd: str) -> None:
    print(f"--- {cmd}")
    _stdin, stdout, _stderr = client.exec_command(cmd, timeout=40)
    print(stdout.read().decode("utf-8", errors="replace")[:2000])


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username="ubuntu", password=PASS, timeout=25)
    run(client, "head -30 /home/ubuntu/apps/TradingBot/UtilityServer.py")
    run(client, "tail -8 /home/ubuntu/apps/TradingBot/UtilityServer.py")
    run(
        client,
        r"""python3 - <<'PY'
import requests
tests = [
    ('5500/searchCode', 'http://127.0.0.1:5500/searchCode', {'regex':'00700','market':['HK']}, {}),
    ('5600/getCurrentPrice', 'http://127.0.0.1:5600/getCurrentPrice', {'code':'00700.HK'}, {}),
    ('146.56.225.252:5600/searchCode', 'http://146.56.225.252:5600/searchCode', {'regex':'00700'}, {'Authorization':'Bearer sk-8v9w0x1y2z3a4b5c6d7e8f9g0h1i2j3k4l5m6n7o8p9q0r1s2t3u4v5w6x7y8z9'}),
    ('43.134.94.87:7000/openapi', 'http://43.134.94.87:7000/openapi.json', None, {}),
]
for label, url, body, headers in tests:
    try:
        if body is None:
            r = requests.get(url, headers=headers, timeout=10)
        else:
            r = requests.post(url, json=body, headers={**headers, 'Content-Type':'application/json'}, timeout=15)
        print(label, r.status_code, r.text[:180].replace(chr(10),' '))
    except Exception as e:
        print(label, 'ERR', e)
PY""",
    )
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
