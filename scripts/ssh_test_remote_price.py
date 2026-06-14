#!/usr/bin/env python3
"""Test remote TradingServer get-price vs local bot getCurrentPrice."""

from __future__ import annotations

import os
import sys

import paramiko

PASS = os.environ.get("SSH_PASS", "")
BOT = "118.195.135.97"
TRADE = "43.134.94.87"
MCP = os.environ.get("MCP_TOKEN", "")


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(BOT, username="ubuntu", password=PASS, timeout=25)

    script = f"""
import json, requests, sys
sys.path.insert(0, '/home/ubuntu/apps/TradingBot')
from mcpAPIServer import get_user_id_by_mcp_token

mcp = {MCP!r}
code = '00700.HK'
uid = get_user_id_by_mcp_token(mcp)
print('mcp_token -> user_id:', uid)

tests = []

# 1) 本机 Bot（当前 MCP 转发目标）
try:
    r = requests.post('http://127.0.0.1:5600/getCurrentPrice', json={{'code': code}}, timeout=20)
    tests.append(('local:5600/getCurrentPrice', r.status_code, r.text[:160]))
except Exception as e:
    tests.append(('local:5600/getCurrentPrice', 'ERR', str(e)))

# 2) 远程 TradingServer（富途网关）
if uid:
    for host in ['43.134.94.87:7000', '127.0.0.1:6300']:
        try:
            r = requests.post(
                f'http://{{host}}/v1/futu/get-price',
                json={{'code': code, 'user_id': str(uid)}},
                timeout=20,
            )
            tests.append((f'remote:{{host}}/v1/futu/get-price', r.status_code, r.text[:200]))
        except Exception as e:
            tests.append((f'remote:{{host}}/v1/futu/get-price', 'ERR', str(e)))

# 3) 5900 getTicker（带 mcp_token，走用户绑定 trade host）
try:
    import os
    from Config import APIConnection
    # read market api key from env if any
    r = requests.post(
        'http://127.0.0.1:5700/getTicker',
        json={{'mcp_token': mcp, 'code': code, 'num': 2}},
        headers={{'Authorization': 'Bearer REPLACE_MK'}},
        timeout=20,
    )
    tests.append(('market:5700/getTicker', r.status_code, r.text[:200]))
except Exception as e:
    tests.append(('market:5700/getTicker', 'ERR', str(e)))

print('--- results ---')
for label, status, body in tests:
    print(label, status)
    print(' ', body)
"""
    # inject market api key from local config
    from pathlib import Path
    import json as _json

    root = Path(__file__).resolve().parents[1]
    cfg = _json.loads((root / "config.local.json").read_text(encoding="utf-8"))
    script = script.replace("REPLACE_MK", cfg["api_key"])

    _stdin, stdout, stderr = client.exec_command(f"python3 - <<'PY'\n{script}\nPY", timeout=60)
    print(stdout.read().decode("utf-8", errors="replace"))
    err = stderr.read().decode("utf-8", errors="replace")
    if err.strip():
        print("STDERR:", err[:800])
    client.close()

    # direct from local to remote TradingServer
    print("\n=== 本机直连远程 TradingServer ===")
    import httpx

    with httpx.Client(timeout=30) as http:
        for path in ["/health", "/v1/futu/get-price"]:
            url = f"http://{TRADE}:7000{path}"
            try:
                if path == "/health":
                    r = http.get(url)
                else:
                    r = http.post(url, json={"code": "00700.HK", "user_id": "probe"})
                print(url, r.status_code, r.text[:200])
            except Exception as exc:
                print(url, "ERR", exc)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
