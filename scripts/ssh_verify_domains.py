#!/usr/bin/env python3
"""Verify tool domains deployed on remote server."""

from __future__ import annotations

import os
import sys

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    cmd = r"""
source ~/.profile
test -f ~/.geegoo/geegoo-agent/src/geegoo_agent/tools/domains.py && echo domains_ok
~/.geegoo/geegoo-agent/venv/bin/python <<'PY'
from geegoo_agent.tools.domains import CHAT_ON_DEMAND_TOOLS
print("list_grid_reminders", "list_grid_reminders" in CHAT_ON_DEMAND_TOOLS)
print("no_report_bot_codes", "get_report_bot_codes" not in CHAT_ON_DEMAND_TOOLS)
print("chat_tool_count", len(CHAT_ON_DEMAND_TOOLS))
PY
geegoo doctor 2>&1 | head -15
"""
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=90)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    print(out)
    if err.strip():
        print(err, file=sys.stderr)
    code = stdout.channel.recv_exit_status()
    client.close()
    return code


if __name__ == "__main__":
    raise SystemExit(main())
