#!/usr/bin/env python3
import json
from pathlib import Path

import paramiko

cfg = json.loads(
    Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json").read_text(
        encoding="utf-8"
    )
)
ssh = cfg["targets"]["geegoo-signal"]["ssh"]
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"))

cmd = r"""bash -lc 'cd /root/apps/GeeGooSignal && set -a && source .env && set +a && python3 <<'"'"'PY'"'"'
import json, os, urllib.request
key = os.environ.get("GEEGOO_SIGNAL_SIGNAL_API_KEY", "")

def post(body):
    req = urllib.request.Request(
        "http://127.0.0.1:3200/getDashboardKline",
        data=json.dumps(body).encode(),
        method="POST",
        headers={
            "Content-Type": "application/json",
            "Authorization": "Bearer " + key,
        },
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return json.loads(resp.read().decode())

for lang in ("cn", "en"):
    rows = post({
        "code": "TEST.US",
        "language": lang,
        "frequency": "daily",
        "pattern_list": ["Doji"],
        "bars": [{"open": 10, "high": 12, "low": 8, "close": 10.05}],
    })
    signal = ((rows[0].get("signal") or [{}])[0]) if rows else {}
    print(lang, signal.get("info", ""))
PY'"""

_, stdout, stderr = client.exec_command(cmd, timeout=180)
print((stdout.read() + stderr.read()).decode("utf-8", "replace"))
