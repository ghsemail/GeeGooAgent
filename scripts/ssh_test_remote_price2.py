#!/usr/bin/env python3
"""Test remote get-price with user's trade_api_token."""

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
    script = f"""
import sys, requests, json
sys.path.insert(0, '/home/ubuntu/apps/TradingBot')
from bson import ObjectId
from mcpAPIServer import get_user_id_by_mcp_token
from Config import DBConnection

uid = get_user_id_by_mcp_token({MCP!r})
user = DBConnection.db.user.find_one({{'_id': ObjectId(uid)}}, {{'trade': 1}})
trade = (user or {{}}).get('trade', {{}})
print('user_id:', uid)
print('trade.bot_host:', trade.get('bot_host'))
print('trade.bot_port:', trade.get('bot_port'))
print('has trade_api_token:', bool(trade.get('trade_api_token')))

host = trade.get('bot_host') or '43.134.94.87'
port = trade.get('bot_port') or 7000
token = (trade.get('trade_api_token') or '').strip()
url = f"http://{{host}}:{{port}}/v1/futu/get-price"
headers = {{'Content-Type': 'application/json'}}
if token:
    headers['x-trading-token'] = token
body = {{'code': '00700.HK', 'user_id': str(uid)}}
r = requests.post(url, json=body, headers=headers, timeout=25)
print('remote get-price:', url)
print('HTTP', r.status_code)
print(r.text[:400])

# same path via FutuTrade helper
from Config import FutuTrade
try:
    price = FutuTrade.getPrice('00700.HK', user_id=uid)
    print('FutuTrade.getPrice with user_id:', price)
except Exception as e:
    print('FutuTrade.getPrice ERR:', e)
"""
    _i, o, e = client.exec_command(f"python3 - <<'PY'\n{script}\nPY", timeout=60)
    print(o.read().decode("utf-8", errors="replace"))
    err = e.read().decode("utf-8", errors="replace")
    if err.strip():
        print("STDERR", err[:500])
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
