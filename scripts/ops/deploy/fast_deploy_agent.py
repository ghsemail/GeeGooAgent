#!/usr/bin/env python3
"""Fast agent deploy: git pull + go build only (no service restart)."""
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-agent"]["ssh"]
cmd = (
    "cd ~/.geegoo/geegoo-agent && git fetch origin main && git reset --hard origin/main && "
    "go build -o geegoo ./cmd/geegoo && "
    "cp -f geegoo ~/.geegoo/bin/geegoo.bin && "
    "go build -o ~/.geegoo/bin/agentRuntimeServer ./cmd/agent-runtime && "
    "git rev-parse --short HEAD && ls -la geegoo ~/.geegoo/bin/geegoo.bin"
)
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=30)
_, o, e = c.exec_command(cmd, timeout=600)
print((o.read() + e.read()).decode("utf-8", "replace"))
c.close()
