#!/usr/bin/env python3
"""Deploy GeeGooAgent scheduler fix to production."""
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE_DIR = "/home/ubuntu/.geegoo/geegoo-agent"

cmds = [
    f"cd {REMOTE_DIR} && git fetch origin main && git reset --hard origin/main",
    f"cd {REMOTE_DIR} && bash start.sh restart-all",
    "sleep 3",
    f"cd {REMOTE_DIR} && bash start.sh status",
    "curl -s http://127.0.0.1:3400/v1/scheduler/status",
    "curl -s http://127.0.0.1:3400/health",
    "ps aux | grep -E 'agentRuntimeServer|scheduler run' | grep -v grep",
    "tail -n 5 /home/ubuntu/.geegoo/geegoo-agent/scheduler.out 2>/dev/null || echo no_scheduler_log",
    "cat /home/ubuntu/.geegoo/data/scheduler/jobs.json 2>/dev/null || echo no_jobs_json",
]

cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
for cmd in cmds:
    print("===", cmd[:100])
    _, o, e = c.exec_command(cmd, timeout=300)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    print(out[:6000])
    print()
c.close()
