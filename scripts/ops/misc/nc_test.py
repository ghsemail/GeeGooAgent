#!/usr/bin/env python3
import json, paramiko
from pathlib import Path
D=Path(r"C:/Users/ghsemail/.cursor/skills/remote-deploy/deploy.json")
s=json.loads(D.read_text(encoding="utf-8-sig"))["targets"]["geegoo-bot"]["ssh"]
c=paramiko.SSHClient(); c.set_missing_host_key_policy(paramiko.AutoAddPolicy()); c.connect(s["host"], username=s["user"], password=s["password"], timeout=60)
_,o,_=c.exec_command("nc -zv -w 8 82.157.97.76 3300 2>&1; echo exit:$?", timeout=20)
print(o.read().decode())
c.close()
