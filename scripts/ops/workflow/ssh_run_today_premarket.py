#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
PROBE = Path(__file__).resolve().parent / "run_today_premarket_remote.py"
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
c = paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(hostname=ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
_, o, e = c.exec_command(f"python3 <<'PY'\n{PROBE.read_text(encoding='utf-8')}\nPY", timeout=1900)
print(o.read().decode())
if e.read().decode().strip():
    print("ERR", e.read().decode())
c.close()
