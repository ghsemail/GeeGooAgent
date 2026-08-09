#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
PROBE = Path(__file__).resolve().parent / "probe_agent_sessions.py"
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(hostname=ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
_, o, _ = client.exec_command(f"python3 <<'PY'\n{PROBE.read_text(encoding='utf-8')}\nPY", timeout=60)
print(o.read().decode())
client.close()
