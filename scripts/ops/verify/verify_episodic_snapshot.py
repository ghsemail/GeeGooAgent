#!/usr/bin/env python3
"""Smoke: one chat turn should create episodic row."""
from __future__ import annotations
import json, subprocess, urllib.request
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
key=env.get('GEEGOO_AGENT_RUNTIME_API_KEY','')
mcp=json.load(open('/home/ubuntu/.geegoo/config.json')).get('mcp_token','')
headers={'Content-Type':'application/json','X-MCP-Token':mcp,'X-Approve-Writes':'true'}
if key: headers['Authorization']='Bearer '+key
body=json.dumps({'message':'只说ok','session_id':'','mcp_token':mcp}).encode()
req=urllib.request.Request('http://127.0.0.1:3400/v1/chat/stream', data=body, method='POST', headers=headers)
raw=urllib.request.urlopen(req, timeout=120).read().decode('utf-8','replace')
print('episode_snapshot' in raw, 'distill' in raw)

p=subprocess.run(['bash','-lc','tr "\\0" "\\n" < /proc/$(pgrep -f agentRuntimeServer | head -1)/environ | grep ^GEEGOO_PG_DSN='], capture_output=True, text=True)
dsn=p.stdout.strip().split('=',1)[1]
print(subprocess.run(['psql', dsn, '-c', 'SELECT COUNT(*) FROM agent_episodes'], capture_output=True, text=True).stdout)
print(subprocess.run(['psql', dsn, '-c', 'SELECT id, left(summary,60) FROM agent_episodes ORDER BY id DESC LIMIT 2'], capture_output=True, text=True).stdout)
"""

cfg=json.loads(DEPLOY.read_text(encoding='utf-8'))
s=cfg['targets']['geegoo-agent']['ssh']
c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s['host'], username=s['user'], password=s.get('password'), timeout=30)
_,o,e=c.exec_command("python3 <<'PY'\n"+REMOTE+"\nPY", timeout=180)
print(o.read().decode())
c.close()
