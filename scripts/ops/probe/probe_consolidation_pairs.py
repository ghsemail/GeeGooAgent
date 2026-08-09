#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r"""
import json, subprocess, os

p=subprocess.run(['bash','-lc','tr "\\0" "\\n" < /proc/$(pgrep -f agentRuntimeServer | head -1)/environ | grep GEEGOO_PG_DSN'], capture_output=True, text=True)
dsn=p.stdout.strip().split('=',1)[1] if '=' in p.stdout else ''
if not dsn:
    raise SystemExit('no dsn')

sql = '''
SELECT id, message_count, left(COALESCE(summary,''),60) AS summary,
       metadata->>'memory_consolidated_pairs' AS consolidated_pairs
FROM agent_chat_sessions
ORDER BY updated_at DESC
LIMIT 8;
'''
print(subprocess.run(['psql', dsn, '-c', sql], capture_output=True, text=True).stdout)

# count user/assistant pairs in latest session messages (rough)
sql2 = '''
SELECT session_id, role, left(content,40)
FROM agent_chat_messages
WHERE session_id = (SELECT id FROM agent_chat_sessions ORDER BY updated_at DESC LIMIT 1)
ORDER BY id;
'''
print('--- latest session messages ---')
print(subprocess.run(['psql', dsn, '-c', sql2], capture_output=True, text=True).stdout)
"""

cfg=json.loads(DEPLOY.read_text(encoding='utf-8'))
s=cfg['targets']['geegoo-agent']['ssh']
c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
_,o,e=c.exec_command("python3 <<'PY'\n"+REMOTE+"\nPY", timeout=60)
print(o.read().decode())
err=e.read().decode()
if err.strip(): print(err)
c.close()
