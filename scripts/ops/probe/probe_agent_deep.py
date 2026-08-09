#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
cmds = [
    "ps aux | grep -E 'geegoo|scheduler' | grep -v grep",
    "systemctl list-units --type=service 2>/dev/null | grep -i geegoo || true",
    "crontab -l 2>/dev/null || echo no_crontab",
    "ls -la /home/ubuntu/.geegoo/geegoo-agent/scheduler/ 2>/dev/null; cat /home/ubuntu/.geegoo/geegoo-agent/scheduler/jobs.json 2>/dev/null || echo no_jobs_json",
    "ls -la /home/ubuntu/.geegoo/bin/ 2>/dev/null",
    "cat /home/ubuntu/.geegoo/geegoo-agent/config.json 2>/dev/null | head -40",
    "find /home/ubuntu/.geegoo -name 'start.sh' 2>/dev/null",
    "ls -laR /home/ubuntu/.geegoo/geegoo-agent/reports 2>/dev/null | head -30",
    "journalctl -u geegoo-agent* --no-pager -n 20 2>/dev/null || true",
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
