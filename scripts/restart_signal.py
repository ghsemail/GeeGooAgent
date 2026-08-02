#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
sig = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(sig["host"], username=sig["user"], password=sig.get("password"))
_, o, e = c.exec_command("cd /root/apps/GeeGooSignal && bash start.sh restart 2>&1 | tail -15", timeout=300)
print((o.read()+e.read()).decode())
