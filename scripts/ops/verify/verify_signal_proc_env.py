#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))

cmd = r"""bash -lc 'cd /root/apps/GeeGooSignal && set -a && source .env && set +a
echo ".env PROVIDER=$GEEGOO_DATA_PROVIDER LOCAL=$GEEGOO_DATA_MARKET_LOCAL"
pid=$(pgrep -f "signal-api" | head -1)
if [ -n "$pid" ]; then
  echo "signal-api pid=$pid"
  tr "\0" "\n" < /proc/$pid/environ | grep -E "GEEGOO_DATA_(PROVIDER|MARKET_LOCAL|HTTP)" || true
fi
'"""
_, o, e = c.exec_command(cmd, timeout=30)
print((o.read()+e.read()).decode("utf-8", "replace"))
