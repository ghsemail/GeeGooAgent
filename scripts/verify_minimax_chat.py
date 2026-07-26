#!/usr/bin/env python3
"""Verify MiniMax catalog model works after /v1 URL fix."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
MINIMAX_ID = "69b64f9904f91361223f8e51"

REMOTE = rf"""
import json, os, urllib.request

cfg = json.loads(open('/home/ubuntu/.geegoo/config.json').read())
env = {{}}
for line in open('/home/ubuntu/.geegoo/agent.env'):
    line=line.strip()
    if line and not line.startswith('#') and '=' in line:
        k,v=line.split('=',1); env[k]=v.strip().strip('"').strip("'")
runtime_key = env.get('GEEGOO_AGENT_RUNTIME_API_KEY','')
mcp = cfg.get('mcp_token','')
body = json.dumps({{
    'message': '只回复 ok',
    'session_id': '',
    'mcp_token': mcp,
}}).encode()
headers = {{
    'Content-Type': 'application/json',
    'X-MCP-Token': mcp,
    'X-Approve-Writes': 'true',
}}
if runtime_key:
    headers['Authorization'] = f'Bearer {{runtime_key}}'
# switch user settings to MiniMax
settings_dir = '/home/ubuntu/.geegoo/geegoo-agent/user_llm_settings'
os.makedirs(settings_dir, exist_ok=True)
# use a test user file; runtime resolves by X-User-Id when present
user_id = 'minimax-smoke-test'
open(f'{{settings_dir}}/{{user_id}}.json','w',encoding='utf-8').write(json.dumps({{
    'catalog_model_id': '{MINIMAX_ID}',
    'use_ops_model': True,
    'thinking': 'off',
}}, indent=2))
headers['X-User-Id'] = user_id
req = urllib.request.Request('http://127.0.0.1:3400/v1/chat/stream', data=body, headers=headers, method='POST')
raw = urllib.request.urlopen(req, timeout=120).read().decode('utf-8','replace')
if 'error' in raw and 'turn_end' not in raw:
    print(raw[-800:])
    raise SystemExit(1)
if '404' in raw and 'nginx' in raw:
    print('STILL 404')
    print(raw[-800:])
    raise SystemExit(1)
print('OK minimax stream length', len(raw))
for line in raw.split('\n'):
    if line.startswith('data:') and 'assistant_text' in line:
        print(line[:200])
        break
"""


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(f"python3 <<'PY'\n{REMOTE}\nPY", timeout=180)
    print(o.read().decode())
    err = e.read().decode()
    if err.strip():
        print(err)
    c.close()


if __name__ == "__main__":
    main()
