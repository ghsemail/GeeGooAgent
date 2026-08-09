#!/usr/bin/env python3
"""List catalog models and probe their LLM base URLs."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r"""
import json, urllib.error, urllib.request

cfg = json.loads(open('/home/ubuntu/.geegoo/config.json').read())
cat_key = (cfg.get('signal_catalog_api_key') or cfg.get('signal_api_key') or '').strip()
base = cfg.get('signal_base_url', 'http://146.56.225.252:3210').rstrip('/')
req = urllib.request.Request(
    base + '/getModel',
    data=b'{}',
    headers={'Content-Type': 'application/json', 'Authorization': f'Bearer {cat_key}'},
    method='POST',
)
models = json.loads(urllib.request.urlopen(req, timeout=20).read().decode())
for m in models:
    mid = m.get('model_id') or m.get('_id')
    name = m.get('name') or m.get('display_name')
    bu = (m.get('base_url') or '').rstrip('/')
    tok = (m.get('token') or '')[-6:]
    print(f"- {mid} | {name} | base={bu} | token=...{tok}")
    if not bu or not m.get('token'):
        print('  SKIP probe (missing base/token)')
        continue
    for url in (bu + '/chat/completions', bu + '/v1/chat/completions'):
        payload = json.dumps({'model': name, 'messages': [{'role':'user','content':'ok'}], 'max_tokens': 8}).encode()
        r = urllib.request.Request(url, data=payload, headers={'Content-Type':'application/json','Authorization':'Bearer '+m['token']}, method='POST')
        try:
            with urllib.request.urlopen(r, timeout=15) as resp:
                print('  OK', url, resp.status)
                break
        except urllib.error.HTTPError as he:
            body = he.read()[:80].decode('utf-8','replace')
            print('  FAIL', url, he.code, body)
"""


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(f"python3 <<'PY'\n{REMOTE}\nPY", timeout=180)
    print(o.read().decode())
    if e.read().decode().strip():
        print(e.read().decode())
    c.close()


if __name__ == "__main__":
    main()
