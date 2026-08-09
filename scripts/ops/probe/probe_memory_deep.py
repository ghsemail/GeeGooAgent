#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r"""
import json, os, subprocess, urllib.request

def envload():
    e={}
    for line in open('/home/ubuntu/.geegoo/agent.env'):
        line=line.strip()
        if line and not line.startswith('#') and '=' in line:
            k,v=line.split('=',1); e[k]=v.strip().strip('"').strip("'")
    return e

env=envload()
print('GEEGOO_PG_DSN in agent.env:', 'yes' if env.get('GEEGOO_PG_DSN') else 'no')
print('GEEGOO_VECTOR_ENABLE in agent.env:', env.get('GEEGOO_VECTOR_ENABLE','(unset)'))

# start.sh may export vars
for f in ['/home/ubuntu/.geegoo/geegoo-agent/start.sh']:
    try:
        txt=open(f).read()
        for kw in ['GEEGOO_PG_DSN','GEEGOO_VECTOR_ENABLE','GEEGOO_SESSION_STORE']:
            for line in txt.splitlines():
                if kw in line:
                    print('start.sh:', line.strip()[:120])
    except FileNotFoundError:
        pass

# runtime env from pid
p=subprocess.run(['bash','-lc','tr "\\0" "\\n" < /proc/$(pgrep -f agentRuntimeServer | head -1)/environ | grep -E "GEEGOO_PG|GEEGOO_VECTOR|GEEGOO_SESSION"'], capture_output=True, text=True)
print('\n=== runtime process env ===')
print(p.stdout or p.stderr or '(none)')

dsn=''
for line in (p.stdout or '').splitlines():
    if line.startswith('GEEGOO_PG_DSN='):
        dsn=line.split('=',1)[1]
if not dsn:
    dsn=env.get('GEEGOO_PG_DSN','')

if dsn:
    for label, sql in [
        ('chunks', "SELECT id, source, (embedding IS NOT NULL) AS has_vec, left(content,80) FROM agent_memory_chunks ORDER BY id"),
        ('episodes', "SELECT id, left(summary,100), user_id FROM agent_episodes ORDER BY id DESC LIMIT 10"),
        ('counts', "SELECT 'chunks' t, count(*) FROM agent_memory_chunks UNION ALL SELECT 'with_vec', count(*) FROM agent_memory_chunks WHERE embedding IS NOT NULL UNION ALL SELECT 'episodes', count(*) FROM agent_episodes"),
    ]:
        print('\n===', label, '===')
        r=subprocess.run(['psql', dsn, '-c', sql], capture_output=True, text=True)
        print(r.stdout or r.stderr)
else:
    print('no dsn for psql')

key=env.get('GEEGOO_AGENT_RUNTIME_API_KEY','')
h={'Authorization':'Bearer '+key} if key else {}
dash=json.loads(urllib.request.urlopen(urllib.request.Request('http://127.0.0.1:3400/v1/dashboard/data', headers=h), timeout=20).read())
print('\n=== dashboard episodes field ===', len(dash.get('episodes') or []))
for ep in (dash.get('episodes') or [])[:3]:
    print(ep)
print('facts', len(dash.get('facts') or []))
for f in (dash.get('facts') or [])[:2]:
    print({k:f.get(k) for k in ['id','source','subject']})
"""

cfg=json.loads(DEPLOY.read_text(encoding='utf-8'))
s=cfg['targets']['geegoo-agent']['ssh']
c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
_,o,e=c.exec_command("python3 <<'PY'\n"+REMOTE+"\nPY", timeout=120)
print(o.read().decode())
if e.read().decode().strip(): print(e.read().decode())
c.close()
