#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))


def run(target: str, cmd: str) -> str:
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    _, o, e = c.exec_command(cmd, timeout=120)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def main() -> int:
    print(run("geegoo-signal", "grep -E 'NEWS_SERVER|CATALOG_API_KEY' /root/apps/GeeGooSignal/.env"))
    print("\n=== catalog delete with real run_id ===")
    print(
        run(
            "geegoo-signal",
            '''python3 - <<'PY'
import json, urllib.request, os
key = open('/root/apps/GeeGooSignal/.env').read().split('GEEGOO_SIGNAL_CATALOG_API_KEY=')[1].split('\\n')[0].strip()
logs = json.load(urllib.request.urlopen(urllib.request.Request(
    'http://127.0.0.1:3210/getNewsRefreshLogs',
    data=json.dumps({'limit': 1}).encode(),
    headers={'Content-Type': 'application/json', 'Authorization': 'Bearer '+key},
    method='POST',
)))
print('logs', len(logs))
if not logs:
    raise SystemExit('no logs')
rid = logs[0]['run_id']
print('run_id', rid)
req = urllib.request.Request(
    'http://127.0.0.1:3210/deleteNewsRefreshLogs',
    data=json.dumps({'run_id': rid}).encode(),
    headers={'Content-Type': 'application/json', 'Authorization': 'Bearer '+key},
    method='POST',
)
print('delete', urllib.request.urlopen(req).read().decode())
PY''',
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
