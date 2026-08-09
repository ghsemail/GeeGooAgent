#!/usr/bin/env python3
import json, urllib.request

BASE = "http://146.56.225.252:8088"

# try via agent server directly
import paramiko
from pathlib import Path
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
s = cfg["targets"]["geegoo-agent"]["ssh"]
c = paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s["host"], username=s["user"], password=s["password"])
for node in ("ashare-cn", "us-hk"):
    _, o, _ = c.exec_command(f"curl -s http://127.0.0.1:3400/v1/data/nodes/{node}/news/health", timeout=60)
    raw = o.read().decode()
    print(f"=== {node} ===")
    try:
        d = json.loads(raw)
        sources = d.get("sources", [])
        print("sources count", len(sources))
        for item in sources[:12]:
            print(item)
    except Exception as e:
        print(raw[:500], e)
c.close()
