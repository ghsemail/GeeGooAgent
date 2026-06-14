#!/usr/bin/env python3
"""Inspect TradingServer on Signal host 43.134.94.87:7000."""

from __future__ import annotations

import json
import os
import sys

import paramiko

PASS = os.environ.get("SSH_PASS", "")
HOST = "43.134.94.87"


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username="ubuntu", password=PASS, timeout=25)
    cmd = r"""python3 - <<'PY'
import json, urllib.request
spec = json.load(urllib.request.urlopen('http://127.0.0.1:7000/openapi.json', timeout=10))
paths = sorted(spec.get('paths', {}).keys())
print('TradingServer paths (sample):')
for p in paths[:40]:
    print(' ', p)
print('total paths:', len(paths))
PY"""
    _stdin, stdout, _stderr = client.exec_command(cmd, timeout=30)
    print(stdout.read().decode())
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
