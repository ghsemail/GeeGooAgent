#!/usr/bin/env python3
"""Fix GeeGooSignal hybrid local routing and restart signal API."""
import json
import re
import time
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))

def run(cmd, timeout=120):
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", "replace")
    print(out)
    return out

# Ensure remote-only data provider (HK/CN local OpenD not wired in Go signal-api)
run(r"""bash -lc 'cd /root/apps/GeeGooSignal
set -a; source .env; set +a
grep -E "^GEEGOO_DATA_(PROVIDER|MARKET_LOCAL)=" .env || true
python3 <<PY
from pathlib import Path
p = Path(".env")
text = p.read_text(encoding="utf-8")
lines = text.splitlines()
kv = {}
order = []
for line in lines:
    if not line.strip() or line.lstrip().startswith("#") or "=" not in line:
        order.append(line)
        continue
    k, _, v = line.partition("=")
    kv[k] = v
    order.append(k)
kv["GEEGOO_DATA_PROVIDER"] = "remote"
kv["GEEGOO_DATA_MARKET_LOCAL"] = ""
seen = set()
new_lines = []
for item in order:
    if item in kv and item not in seen:
        new_lines.append(f"{item}={kv[item]}")
        seen.add(item)
    elif "=" not in item and item not in kv:
        new_lines.append(item)
for k, v in kv.items():
    if k not in seen:
        new_lines.append(f"{k}={v}")
p.write_text("\n".join(new_lines) + "\n", encoding="utf-8")
print("updated .env")
PY
grep -E "^GEEGOO_DATA_(PROVIDER|MARKET_LOCAL)=" .env
'""")

run(r"""bash -lc 'cd /root/apps/GeeGooSignal && bash start.sh restart signal-api 2>&1 | tail -20'""", timeout=180)
time.sleep(3)

run(r"""bash -lc 'pid=$(pgrep -f signalAPIServer | head -1); echo pid=$pid; tr "\0" "\n" < /proc/$pid/environ | grep GEEGOO_DATA | sort'""")

c.close()
print("\n--- Re-run user codes verification ---")
