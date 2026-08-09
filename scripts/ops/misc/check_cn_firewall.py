#!/usr/bin/env python3
import json, paramiko
from pathlib import Path
cfg=json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding='utf-8-sig'))
s=cfg['targets']['geegoo-data-cn']['ssh']
c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s['host'],username=s['user'],password=s['password'],timeout=60)
for cmd in [
 'sudo ufw status 2>/dev/null || echo no-ufw',
 'ss -tlnp | grep 3300',
 'curl -s -m 3 http://127.0.0.1:3300/health',
 'curl -s -m 3 http://82.157.97.76:3300/health',
]:
 _,o,_=c.exec_command(cmd, timeout=30)
 print('>>>',cmd); print(o.read().decode())
c.close()
