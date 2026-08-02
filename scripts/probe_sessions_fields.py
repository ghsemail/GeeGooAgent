#!/usr/bin/env python3
import json
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def get_ghsemail_user_id() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    sc = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(sc["host"], username=sc["user"], password=sc.get("password"), timeout=60)
    py = r"""
import json, subprocess
mongo = next(n for n in subprocess.check_output("docker ps --format '{{.Names}}'", shell=True, text=True).splitlines() if 'mongo' in n.lower())
raw = subprocess.check_output(['docker','exec',mongo,'mongosh','Signal_DB','--quiet','--eval',
 "JSON.stringify(db.admin.findOne({username:'ghsemail'},{_id:1})._id)"], text=True).strip()
print(raw)
"""
    _, o, _ = c.exec_command(f"python3 <<'PY'\n{py}\nPY", timeout=60)
    raw = o.read().decode().strip()
    c.close()
    if raw.startswith("{"):
        return json.loads(raw).get("$oid", "")
    return raw.strip('"')


def main() -> None:
    uid = get_ghsemail_user_id()
    print("uid:", uid[:12])
    req = urllib.request.Request(
        "http://146.56.225.252:8088/op_agent/v1/sessions?limit=8",
        headers={
            "Authorization": f"Bearer {KEY}",
            "X-User-Id": uid,
            "X-Client-Source": "trading_operation",
        },
    )
    with urllib.request.urlopen(req, timeout=30) as r:
        data = json.loads(r.read().decode())
    for s in data.get("sessions", [])[:8]:
        print(
            {
                "id": s.get("id"),
                "user_id": (s.get("user_id") or "")[:12],
                "username": s.get("username"),
                "source": s.get("source"),
                "title": (s.get("title") or "")[:30],
            }
        )


if __name__ == "__main__":
    main()
