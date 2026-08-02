#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))
_, o, e = c.exec_command("tr '\\0' '\\n' < /proc/569497/environ | grep -E 'GEEGOO_DATA|GEEGOO_SIGNAL_MONGO' | sort", timeout=30)
print((o.read()+e.read()).decode("utf-8", "replace"))
