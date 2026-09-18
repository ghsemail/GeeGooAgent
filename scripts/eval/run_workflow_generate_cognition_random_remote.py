#!/usr/bin/env python3
"""Upsert workflow eval cases and run random strategy cognition smoke on geegoo-agent."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parent
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE_DIR = "/tmp/geegoo_workflow_eval"
UPSERT_SQL = REMOTE_DIR + "/workflow_eval_seed.sql"


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        ssh["host"],
        port=int(ssh.get("port", 22)),
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=30,
    )
    client.exec_command(f"mkdir -p {REMOTE_DIR}")
    sftp = client.open_sftp()
    sftp.put(str(ROOT / "run_workflow_generate_cognition_random_live.py"), f"{REMOTE_DIR}/run_workflow_generate_cognition_random_live.py")
    sftp.put(str(ROOT / "workflow_eval_seed.sql"), UPSERT_SQL)
    sftp.close()

    upsert_cmd = (
        f"set -a && source ~/.geegoo/agent.env && set +a && "
        f'psql "$GEEGOO_PG_DSN" -v ON_ERROR_STOP=1 -f {UPSERT_SQL}'
    )
    _, stdout, stderr = client.exec_command(upsert_cmd, timeout=120)
    upsert_out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    if upsert_out.strip():
        print("upsert:", upsert_out.strip())

    cmd = f"cd {REMOTE_DIR} && python3 run_workflow_generate_cognition_random_live.py"
    if len(sys.argv) > 1:
        cmd += " " + " ".join(json.dumps(a) for a in sys.argv[1:])
    _, stdout, stderr = client.exec_command(cmd, timeout=960)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    print(out)
    if err.strip():
        print("STDERR:", err)
    client.close()
    return 0 if "SUMMARY: PASS" in out else 1


if __name__ == "__main__":
    raise SystemExit(main())
