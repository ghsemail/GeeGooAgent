#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-data"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))

script = r"""bash -lc 'cd /root/apps/GeeGooData && python3 <<PY
import json, subprocess
for op in [
  {"operation":"klines","code":"600519.SH","frequency":"daily","futu_host":"127.0.0.1","futu_port":11111},
  {"operation":"klines","code":"600519.SH","frequency":"5m","futu_host":"127.0.0.1","futu_port":11111},
]:
  p=subprocess.run(["python3","scripts/futu_market_helper.py"],input=json.dumps(op),text=True,capture_output=True)
  print(op["frequency"], p.stdout[:200], p.stderr[:120])
PY'"""
_, o, e = c.exec_command(script, timeout=120)
print((o.read() + e.read()).decode("utf-8", "replace"))
