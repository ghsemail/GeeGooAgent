#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    try:
        sid = "chat-2b0c7c47974e"
        cmd = f"curl -s -w '\\nHTTP:%{{http_code}}\\n' http://127.0.0.1:3400/v1/sessions/{sid}/trace"
        _, o, _ = c.exec_command(cmd, timeout=30)
        print(o.read().decode("utf-8", errors="replace"))
        # check raw step_records_json type in db
        remote = r"""
import subprocess, json
p=subprocess.run(['bash','-lc','tr "\\0" "\\n" < /proc/$(pgrep -f agentRuntimeServer | head -1)/environ | grep ^GEEGOO_PG_DSN='], capture_output=True, text=True)
dsn=p.stdout.strip().split('=',1)[1]
sql = "SELECT id, jsonb_typeof(step_records_json), length(step_records_json::text), left(step_records_json::text, 200) FROM chat_sessions WHERE id='chat-2b0c7c47974e';"
print(subprocess.run(['psql', dsn, '-c', sql], capture_output=True, text=True).stdout)
"""
        _, o2, _ = c.exec_command("python3 <<'PY'\n" + remote + "\nPY", timeout=30)
        print(o2.read().decode("utf-8", errors="replace"))
    finally:
        c.close()


if __name__ == "__main__":
    main()
