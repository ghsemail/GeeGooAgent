#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
cmds = [
    "cat /home/ubuntu/.geegoo/config.json 2>/dev/null | python3 -m json.tool 2>/dev/null | head -50",
    "cat /home/ubuntu/.geegoo/agent.env 2>/dev/null | grep -v KEY | grep -v SECRET | grep -v TOKEN",
    "ls -la /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.out /home/ubuntu/.geegoo/geegoo-agent/*.out 2>/dev/null",
    "wc -l /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.out 2>/dev/null",
    "/home/ubuntu/.geegoo/bin/geegoo scheduler list 2>&1 | head -20",
]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
for cmd in cmds:
    _, o, e = c.exec_command(cmd, timeout=30)
    print("===", cmd[:90])
    print((o.read() + e.read()).decode("utf-8", errors="replace")[:5000])
    print()
c.close()
