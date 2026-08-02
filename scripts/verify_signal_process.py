#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

cfg = json.loads(Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))

cmd = r"""bash -lc 'ps aux | grep -E "signal-api|GeeGooSignal" | grep -v grep
for pid in $(pgrep -f "cmd/signal-api|signal-api"); do
  echo "--- pid $pid cmdline ---"
  tr "\0" " " < /proc/$pid/cmdline; echo
  echo "environ sample:"
  tr "\0" "\n" < /proc/$pid/environ | grep GEEGOO | sort
done
head -30 /root/apps/GeeGooSignal/start.sh
'"""
_, o, e = c.exec_command(cmd, timeout=30)
print((o.read()+e.read()).decode("utf-8", "replace"))
