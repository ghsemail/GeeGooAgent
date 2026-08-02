#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
cmds = [
    "ps aux | grep -E 'geegoo|agent-runtime|scheduler' | grep -v grep | head -10",
    "ls -la /home/ubuntu/.geegoo/geegoo-agent/*.out 2>/dev/null | tail -5",
    "tail -n 40 /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.out 2>/dev/null | grep -iE 'pre_market|post_market|error|fail|scheduler' || tail -n 15 /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.out 2>/dev/null",
    "curl -s -m 5 http://127.0.0.1:3400/health || echo health_fail",
]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
for cmd in cmds:
    _, o, e = c.exec_command(cmd, timeout=30)
    print("===", cmd[:70])
    print((o.read() + e.read()).decode("utf-8", errors="replace")[:2500])
    print()
c.close()
