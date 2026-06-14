#!/usr/bin/env python3
"""Check bot getCurrentPrice error on 118.195.135.97."""

from __future__ import annotations

import os
import sys

import paramiko

PASS = os.environ.get("SSH_PASS", "")


def main() -> int:
    if not PASS:
        return 1
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect("118.195.135.97", username="ubuntu", password=PASS, timeout=25)
    cmds = [
        "tail -30 /home/ubuntu/apps/TradingBot/*.log 2>/dev/null | tail -20",
        "ls -lt /home/ubuntu/apps/TradingBot/*.out 2>/dev/null | head -5",
        "tail -20 /home/ubuntu/apps/TradingBot/botAPIServer.out 2>/dev/null || tail -20 /home/ubuntu/apps/TradingBot/nohup.out 2>/dev/null || echo no log",
        r"""python3 - <<'PY'
import requests
r = requests.post('http://127.0.0.1:5600/getCurrentPrice', json={'code':'00700.HK'}, timeout=20)
print('status', r.status_code)
print(r.text[:500])
PY""",
    ]
    for c in cmds:
        print("---", c[:70])
        _i, o, e = client.exec_command(c, timeout=30)
        print(o.read().decode()[:1500])
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
