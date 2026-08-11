#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-data"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=60)
_, o, e = c.exec_command(
    "cd /root/apps/GeeGooData && git fetch origin main && git reset --hard origin/main && bash start.sh restart",
    timeout=120,
)
print((o.read() + e.read()).decode("utf-8", "replace"))
c.close()
