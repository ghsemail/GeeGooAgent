#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
s = cfg["targets"]["geegoo-agent"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
_, o, _ = c.exec_command("cat /home/ubuntu/.geegoo/config.json", timeout=30)
data = json.loads(o.read().decode("utf-8", errors="replace"))
for k in sorted(data):
    if "url" in k.lower() or "key" in k.lower() or k in ("mcp_token",):
        print(k, "=", data[k])
c.close()
