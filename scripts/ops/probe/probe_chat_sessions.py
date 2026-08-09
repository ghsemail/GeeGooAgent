#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r"""
import json, subprocess
p=subprocess.run(['bash','-lc','tr "\\0" "\\n" < /proc/$(pgrep -f agentRuntimeServer | head -1)/environ | grep ^GEEGOO_PG_DSN='], capture_output=True, text=True)
dsn=p.stdout.strip().split('=',1)[1]
sql = '''
SELECT id, left(summary,70) AS summary,
       metadata_json->>'memory_consolidated_pairs' AS consolidated_pairs,
       jsonb_array_length(messages_json) AS msg_count
FROM chat_sessions
ORDER BY updated_at DESC
LIMIT 6;
'''
out=subprocess.run(['psql', dsn, '-At', '-F', '|', '-c', sql], capture_output=True, text=True)
print(out.stdout or out.stderr)
for line in (out.stdout or '').splitlines():
    sid, summary, pairs, msg_count = (line.split('|') + ['','','',''])[:4]
    if not sid:
        continue
    # count user-assistant pairs from messages_json
    q = f"SELECT messages_json FROM chat_sessions WHERE id='{sid}'"
    r=subprocess.run(['psql', dsn, '-At', '-c', q], capture_output=True, text=True)
    try:
        msgs=json.loads(r.stdout.strip() or '[]')
    except Exception:
        msgs=[]
    pairs_n=0
    pending=False
    for m in msgs:
        role=m.get('role')
        if role=='user':
            pending=True
        elif role=='assistant' and pending:
            pairs_n+=1
            pending=False
    print(f'session {sid}: pairs={pairs_n} consolidated_at={pairs or 0} msgs={msg_count} summary_len={len(summary)}')
"""

cfg=json.loads(DEPLOY.read_text(encoding='utf-8'))
s=cfg['targets']['geegoo-agent']['ssh']
c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
_,o,e=c.exec_command("python3 <<'PY'\n"+REMOTE+"\nPY", timeout=60)
print(o.read().decode())
c.close()
