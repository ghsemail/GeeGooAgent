#!/usr/bin/env python3
"""Inspect agent LLM config and probe catalog queryModel."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r"""
import json, urllib.error, urllib.request
from pathlib import Path

cfg = json.loads(Path('/home/ubuntu/.geegoo/config.json').read_text(encoding='utf-8'))
llm = cfg.get('llm', {})
print('config.llm:', json.dumps({
    'provider': llm.get('provider'),
    'model': llm.get('model'),
    'base_url': llm.get('base_url'),
    'use_ops_model': llm.get('use_ops_model'),
    'catalog_model_id': llm.get('catalog_model_id'),
    'token_suffix': (llm.get('token_key') or '')[-6:],
}, ensure_ascii=False, indent=2))

cat_key = (cfg.get('signal_catalog_api_key') or cfg.get('signal_api_key') or '').strip()
base = cfg.get('signal_base_url', 'http://146.56.225.252:3210').rstrip('/')
for label, body in [
    ('configured', {'type': 'configured'}),
    ('deepseek-v4-pro', {'model_id': 'deepseek-v4-pro'}),
]:
    req = urllib.request.Request(
        base + '/queryModel',
        data=json.dumps(body).encode(),
        headers={'Content-Type': 'application/json', 'Authorization': f'Bearer {cat_key}'},
        method='POST',
    )
    try:
        doc = json.loads(urllib.request.urlopen(req, timeout=15).read().decode())
    except Exception as e:
        print(f'queryModel {label} error:', e)
        continue
    print(f'queryModel {label}:', json.dumps({
        'name': doc.get('name'),
        'display_name': doc.get('display_name'),
        'base_url': doc.get('base_url'),
        'provider': doc.get('provider'),
        'token_suffix': (doc.get('token') or '')[-6:],
    }, ensure_ascii=False))

model = llm.get('model') or 'deepseek-chat'
base_url = (llm.get('base_url') or 'https://api.deepseek.com').rstrip('/')
token = (llm.get('token_key') or '').strip()
for suffix in ('/chat/completions', '/v1/chat/completions'):
    url = base_url + suffix if not base_url.endswith('/chat/completions') else base_url
    if suffix == '/v1/chat/completions' and base_url.endswith('/v1'):
        url = base_url + '/chat/completions'
    elif suffix == '/v1/chat/completions':
        url = base_url + '/v1/chat/completions'
    else:
        url = base_url + '/chat/completions'
    payload = json.dumps({'model': model, 'messages': [{'role': 'user', 'content': 'ok'}], 'max_tokens': 8}).encode()
    req = urllib.request.Request(url, data=payload, headers={'Content-Type': 'application/json', 'Authorization': f'Bearer {token}'}, method='POST')
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            print('probe OK', url, resp.status)
    except urllib.error.HTTPError as he:
        body = he.read()[:120].decode('utf-8', 'replace')
        print('probe FAIL', url, he.code, body)
"""


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(f"python3 <<'PY'\n{REMOTE}\nPY", timeout=90)
    print(o.read().decode())
    err = e.read().decode()
    if err.strip():
        print(err)
    c.close()


if __name__ == "__main__":
    main()
