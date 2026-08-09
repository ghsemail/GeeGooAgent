#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))
_, o, e = c.exec_command("bash -lc 'cd /root/apps/GeeGooSignal && git log -1 --oneline && rg -n \"dashboardKlineFrequencies|Market \\*marketdata\" internal/signal/dashboard/handler.go | head -5'", timeout=60)
print((o.read()+e.read()).decode("utf-8", "replace"))
