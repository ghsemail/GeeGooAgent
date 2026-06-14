#!/usr/bin/env python3
"""Debug MCP getCurrentPrice after deploy."""

from __future__ import annotations

import os
import sys

import paramiko

PASS = os.environ.get("SSH_PASS", "")
MCP = os.environ.get("MCP_TOKEN", "")


def main() -> int:
    if not PASS:
        return 1
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect("118.195.135.97", username="ubuntu", password=PASS, timeout=25)
    cmds = [
        "ss -tlnp | grep 5700 || echo '5700 not listening'",
        "ps aux | grep -E 'mcpAPIServer|uvicorn.*5700' | grep -v grep | head -5",
        "tail -30 /home/ubuntu/apps/TradingBot/mcpapi.out",
        f"""python3 - <<'PY'
import sys
sys.path.insert(0, '/home/ubuntu/apps/TradingBot')
from mcpAPIServer import get_user_id_by_mcp_token
from Config import FutuTrade
uid = get_user_id_by_mcp_token({MCP!r})
print('user_id', uid)
try:
    p = FutuTrade.getPrice('00700.HK', user_id=uid)
    print('FutuTrade.getPrice', p)
except Exception as e:
    print('ERR', type(e).__name__, e)
PY""",
    ]
    for cmd in cmds:
        print("---", cmd[:80])
        _i, o, e = client.exec_command(cmd, timeout=60)
        print(o.read().decode("utf-8", errors="replace")[:2500])
        err = e.read().decode("utf-8", errors="replace")
        if err.strip():
            print("STDERR", err[:800])
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
