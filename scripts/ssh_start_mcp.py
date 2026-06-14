#!/usr/bin/env python3
"""Start mcpAPIServer and capture startup errors."""

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
    cmd = r"""
cd /home/ubuntu/apps/TradingBot
python3 -c "import mcpAPIServer; print('import ok')" 2>&1 | tail -20
echo '--- start ---'
nohup python3 mcpAPIServer.py >> mcpapi.out 2>&1 &
sleep 4
ss -tlnp | grep 5700 || echo down
tail -15 mcpapi.out
"""
    _i, o, e = client.exec_command(cmd, timeout=60)
    print(o.read().decode("utf-8", errors="replace"))
    err = e.read().decode("utf-8", errors="replace")
    if err.strip():
        print("STDERR", err)

    if MCP:
        verify = f"""
python3 - <<'PY'
import re, requests
text = open('/home/ubuntu/apps/TradingBot/mcpAPIServer.py', encoding='utf-8', errors='replace').read()
key = re.search(r"API_KEY\\s*=\\s*['\\\"]([^'\\\"]+)['\\\"]", text).group(1)
r = requests.post('http://127.0.0.1:5700/getCurrentPrice',
    json={{'mcp_token': {MCP!r}, 'code': '00700.HK'}},
    headers={{'Authorization': f'Bearer {{key}}'}}, timeout=20)
print('HTTP', r.status_code, r.text[:200])
PY
"""
        _i, o, e = client.exec_command(verify, timeout=40)
        print(o.read().decode("utf-8", errors="replace"))
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
