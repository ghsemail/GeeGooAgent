#!/usr/bin/env python3
"""Run name enrichment steps on GeeGooSignal server."""
import json
from pathlib import Path

import paramiko

DEPLOY_CFG = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"

cmd = """
cd /root/apps/GeeGooSignal
nohup bash -c '
  .venv-stock-catalog/bin/python3 scripts/stock_catalog_refresh.py --enrich-only --step en > enrich_en_fix.out 2>&1
  .venv-stock-catalog/bin/python3 scripts/stock_catalog_refresh.py --enrich-only --step zh_hk > enrich_zh_hk.out 2>&1
  echo done > enrich_names.done
' > enrich_names.nohup 2>&1 &
echo started
"""

cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(
    hostname=ssh["host"],
    port=int(ssh.get("port", 22)),
    username=ssh["user"],
    password=ssh.get("password"),
    timeout=30,
)
_, stdout, stderr = client.exec_command(cmd)
print(stdout.read().decode() or stderr.read().decode())
client.close()
