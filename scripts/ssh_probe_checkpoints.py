#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
PROBE = Path(__file__).resolve().parent / "probe_checkpoints.py"
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
c = paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(hostname=ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
_, o, _ = c.exec_command(f"python3 <<'PY'\n{PROBE.read_text(encoding='utf-8')}\nPY", timeout=60)
print(o.read().decode())
c.close()
