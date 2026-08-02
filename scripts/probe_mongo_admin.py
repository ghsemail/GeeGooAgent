#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
for name, path in [
    ("geegoo-bot", "/home/ubuntu/apps/GeeGooBot/.env"),
    ("geegoo-signal", "/root/apps/GeeGooSignal/.env"),
]:
    s = cfg["targets"][name]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s["password"], timeout=60)
    _, o, _ = c.exec_command(f"grep MONGO {path} | head -5", timeout=20)
    print(f"=== {name} ===")
    print(o.read().decode())
    c.close()
