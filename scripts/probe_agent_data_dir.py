#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
cmds = [
    "ls -la /home/ubuntu/.geegoo/data/ 2>/dev/null | head -20",
    "ls -la /home/ubuntu/.geegoo/data/scheduler/ 2>/dev/null; cat /home/ubuntu/.geegoo/data/scheduler/jobs.json 2>/dev/null",
    "find /home/ubuntu/.geegoo/data/reports -type f -name '*.md' 2>/dev/null | sort | tail -15",
    "find /home/ubuntu/.geegoo/data -name 'execution*.log' -o -name '*.log' 2>/dev/null | xargs ls -lt 2>/dev/null | head -10",
    "stat /home/ubuntu/.geegoo/data/reports 2>/dev/null || echo no_reports_dir",
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
