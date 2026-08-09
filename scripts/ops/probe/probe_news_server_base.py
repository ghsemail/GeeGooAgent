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

cmds = [
    "grep NEWS /root/apps/GeeGooSignal/.env || echo NO_NEWS_ENV",
    'curl -s -w "\\n5800:%{http_code}" -X POST http://47.80.14.120:5800/deleteNewsRefreshLogs -H "Content-Type: application/json" -d \'{"run_id":"x"}\' | tail -c 200',
    'curl -s -w "\\n3300:%{http_code}" -X POST http://47.80.14.120:3300/deleteNewsRefreshLogs -H "Content-Type: application/json" -d \'{"run_id":"x"}\' | tail -c 200',
    'KEY=$(grep ^GEEGOO_SIGNAL_CATALOG_API_KEY= /root/apps/GeeGooSignal/.env | cut -d= -f2-); curl -s -X POST http://127.0.0.1:3210/deleteNewsRefreshLogs -H "Content-Type: application/json" -H "Authorization: Bearer $KEY" -d \'{"run_id":"x"}\'',
]
for cmd in cmds:
    print("===", cmd[:80])
    _, o, e = c.exec_command(cmd)
    print((o.read() + e.read()).decode())
c.close()
