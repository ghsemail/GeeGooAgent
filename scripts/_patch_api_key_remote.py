#!/usr/bin/env python3
import json
import re
from pathlib import Path

import paramiko

API_KEY = re.search(
    r'API_KEY\s*=\s*["\']([^"\']+)["\']',
    Path(r"D:\Geegoo\TradingBot\mcp\constants.py").read_text(encoding="utf-8"),
).group(1)

HOST, USER, PASS = "119.45.16.112", "ubuntu", "Ghs@2024"
REMOTE = "/home/ubuntu/.geegoo/config.json"

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(HOST, username=USER, password=PASS, timeout=25)
sftp = c.open_sftp()
with sftp.open(REMOTE, "r") as f:
    raw = json.loads(f.read().decode("utf-8"))
raw["api_key"] = API_KEY
raw["geegoo_api_key"] = API_KEY
with sftp.open(REMOTE, "w") as f:
    f.write(json.dumps(raw, indent=2, ensure_ascii=False).encode("utf-8") + b"\n")
sftp.close()

_, o, _ = c.exec_command(
    'export PATH="$HOME/.geegoo/bin:$PATH"; geegoo doctor --skip-llm 2>&1',
    timeout=120,
)
print(o.read().decode())
c.close()
