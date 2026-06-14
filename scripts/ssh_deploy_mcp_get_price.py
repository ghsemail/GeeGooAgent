#!/usr/bin/env python3
"""Deploy mcpAPIServer getCurrentPrice fix to Bot server and restart MCP."""

from __future__ import annotations

import os
import sys
from pathlib import Path

import paramiko

HOST = "118.195.135.97"
USER = "ubuntu"
REMOTE_PATH = "/home/ubuntu/apps/TradingBot/mcpAPIServer.py"
LOCAL_PATH = Path(r"D:\Geegoo\TradingBot\mcpAPIServer.py")


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 60) -> tuple[str, str]:
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    return out, err


def main() -> int:
    password = os.environ.get("SSH_PASS", "")
    mcp_token = os.environ.get("MCP_TOKEN", "")
    if not password:
        print("Set SSH_PASS", file=sys.stderr)
        return 1
    if not LOCAL_PATH.is_file():
        print(f"Missing local file: {LOCAL_PATH}", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=password, timeout=25)

    backup = f"{REMOTE_PATH}.bak_get_price"
    out, err = run(client, f"cp -a {REMOTE_PATH} {backup} && echo backup_ok")
    print(out.strip() or err.strip())

    sftp = client.open_sftp()
    sftp.put(str(LOCAL_PATH), REMOTE_PATH)
    sftp.close()
    print(f"uploaded {LOCAL_PATH.name} -> {REMOTE_PATH}")

    restart_cmd = r"""
set -e
cd /home/ubuntu/apps/TradingBot
pkill -f 'mcpAPIServer' 2>/dev/null || true
sleep 2
nohup python3 mcpAPIServer.py >> mcpapi.out 2>&1 &
sleep 3
ss -tlnp | grep 5700 || true
ps aux | grep mcpAPIServer | grep -v grep | head -2
"""
    out, err = run(client, restart_cmd, timeout=30)
    print(out.strip())
    if err.strip():
        print("restart stderr:", err[:500])

    if mcp_token:
        verify = f"""
python3 - <<'PY'
import requests, json, os
MCP = {mcp_token!r}
# read sk key from local file header
import re
text = open('/home/ubuntu/apps/TradingBot/mcpAPIServer.py', encoding='utf-8', errors='replace').read()
m = re.search(r"API_KEY\\s*=\\s*['\\\"]([^'\\\"]+)['\\\"]", text)
key = m.group(1) if m else ''
url = 'http://127.0.0.1:5700/getCurrentPrice'
r = requests.post(url, json={{'mcp_token': MCP, 'code': '00700.HK'}},
    headers={{'Authorization': f'Bearer {{key}}', 'Content-Type': 'application/json'}}, timeout=25)
print('verify HTTP', r.status_code)
print(r.text[:300])
PY
"""
        out, err = run(client, verify, timeout=40)
        print(out.strip())
        if err.strip():
            print("verify stderr:", err[:500])

    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
