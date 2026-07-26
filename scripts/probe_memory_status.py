#!/usr/bin/env python3
"""Probe agent-runtime memory: semantic vs episodic vs vector."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r"""
import json, os, urllib.request

def load_env(path):
    env = {}
    for line in open(path):
        line = line.strip()
        if not line or line.startswith('#') or '=' not in line:
            continue
        k, v = line.split('=', 1)
        env[k] = v.strip().strip('"').strip("'")
    return env

env = load_env('/home/ubuntu/.geegoo/agent.env')
key = env.get('GEEGOO_AGENT_RUNTIME_API_KEY', '')
headers = {'Authorization': 'Bearer ' + key} if key else {}

def get(path):
    req = urllib.request.Request('http://127.0.0.1:3400' + path, headers=headers)
    with urllib.request.urlopen(req, timeout=20) as r:
        return json.loads(r.read().decode())

print('=== ENV ===')
for k in ['GEEGOO_PG_DSN', 'GEEGOO_SESSION_STORE', 'GEEGOO_VECTOR_ENABLE', 'OPENAI_API_KEY']:
    v = env.get(k, '')
    if 'DSN' in k and v:
        print(k, '= set (hidden)')
    elif 'KEY' in k and v:
        print(k, '= set suffix', v[-6:])
    else:
        print(k, '=', v or '(unset)')

cfg = json.load(open('/home/ubuntu/.geegoo/config.json'))
emb = cfg.get('embedding', {})
print('config.embedding:', json.dumps({k: emb.get(k) for k in ['model', 'token_key', 'dimensions']}, ensure_ascii=False))

print('\n=== /v1/memory/status ===')
print(json.dumps(get('/v1/memory/status'), indent=2, ensure_ascii=False))

print('\n=== /v1/dashboard/data (memory slice) ===')
dash = get('/v1/dashboard/data')
facts = dash.get('facts') or []
episodes = dash.get('episodes') or []
print('facts:', len(facts))
print('episodes:', len(episodes))
if facts[:1]:
    print('sample fact:', json.dumps(facts[0], ensure_ascii=False)[:200])
if episodes[:1]:
    print('sample episode:', json.dumps(episodes[0], ensure_ascii=False)[:200])

print('\n=== PG counts ===')
import subprocess
dsn = env.get('GEEGOO_PG_DSN', '')
if not dsn:
    print('no PG DSN')
else:
    sql = '''
SELECT 'agent_memory_chunks' AS t, COUNT(*) FROM agent_memory_chunks
UNION ALL
SELECT 'chunks_with_embedding', COUNT(*) FROM agent_memory_chunks WHERE embedding IS NOT NULL
UNION ALL
SELECT 'agent_episodes', COUNT(*) FROM agent_episodes;
'''
    cmd = ['psql', dsn, '-At', '-F', '|', '-c', sql]
    p = subprocess.run(cmd, capture_output=True, text=True)
    print(p.stdout or p.stderr)
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
