#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(
    hostname=ssh["host"],
    port=int(ssh.get("port", 22)),
    username=ssh["user"],
    password=ssh.get("password"),
    timeout=60,
)
_, o, e = c.exec_command(
    "docker exec 0cb244428c30 sh -c 'grep -A30 op_catalog /etc/nginx/conf.d/default.conf 2>/dev/null || grep -R op_catalog /etc/nginx 2>/dev/null | head -40'",
    timeout=60,
)
print((o.read() + e.read()).decode())
c.close()
