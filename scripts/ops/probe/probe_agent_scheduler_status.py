#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
cmds = [
    "curl -s http://127.0.0.1:3400/v1/scheduler/status",
    "ls -la /home/ubuntu/.geegoo/geegoo-agent/scheduler/ 2>/dev/null",
    "cat /home/ubuntu/.geegoo/geegoo-agent/scheduler/jobs.yaml 2>/dev/null || echo no_jobs_yaml",
    "find /home/ubuntu/.geegoo/geegoo-agent -name '*.log' -o -name 'scheduler*' 2>/dev/null | head -20",
    "ls -la /home/ubuntu/.geegoo/geegoo-agent/reports/ 2>/dev/null | tail -10",
]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
for cmd in cmds:
    _, o, e = c.exec_command(cmd, timeout=30)
    print("===", cmd[:80])
    print((o.read() + e.read()).decode("utf-8", errors="replace")[:4000])
    print()
c.close()
