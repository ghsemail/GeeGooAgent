#!/usr/bin/env python3
from __future__ import annotations
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE = r"""
import subprocess
p=subprocess.run(['bash','-lc','tr "\\0" "\\n" < /proc/$(pgrep -f agentRuntimeServer | head -1)/environ | grep ^GEEGOO_PG_DSN='], capture_output=True, text=True)
dsn=p.stdout.strip().split('=',1)[1]
print(subprocess.run(['psql', dsn, '-c', "\\dt"], capture_output=True, text=True).stdout)
"""

cfg=json.loads(DEPLOY.read_text(encoding='utf-8'))
s=cfg['targets']['geegoo-agent']['ssh']
c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
_,o,e=c.exec_command("python3 <<'PY'\n"+REMOTE+"\nPY", timeout=60)
print(o.read().decode())
c.close()
