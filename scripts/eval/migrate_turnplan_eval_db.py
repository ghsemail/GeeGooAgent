#!/usr/bin/env python3
"""Apply TurnPlan eval case seeds to Postgres on geegoo-agent host."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
SQL_FILE = Path(__file__).resolve().parents[2] / "internal/infra/pgschema/postgres_eval.sql"
REMOTE_SQL = "/tmp/turnplan_eval_seed.sql"


def extract_turnplan_sql(text: str) -> str:
    lines = []
    for ln in text.splitlines():
        if ln.startswith("DELETE FROM agent_eval_cases WHERE id LIKE 'turn_plan_%'"):
            lines.append(ln)
        elif ln.startswith("DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing'"):
            lines.append(ln)
        elif ln.startswith("INSERT INTO agent_eval_cases") and "turn_plan_" in ln:
            lines.append(ln)
    if len(lines) < 24:
        raise RuntimeError(f"expected >=24 migration lines, got {len(lines)}")
    return "\n".join(lines) + "\n"


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh_cfg = cfg["targets"]["geegoo-agent"]["ssh"]
    migration = extract_turnplan_sql(SQL_FILE.read_text(encoding="utf-8"))

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    sftp = client.open_sftp()
    with sftp.file(REMOTE_SQL, "w") as f:
        f.write(migration)
    sftp.close()

    cmds = [
        f"set -a && source ~/.geegoo/agent.env && set +a && psql \"$GEEGOO_PG_DSN\" -v ON_ERROR_STOP=1 -f {REMOTE_SQL}",
        "set -a && source ~/.geegoo/agent.env && set +a && psql \"$GEEGOO_PG_DSN\" -tAc \"SELECT count(*) FROM agent_eval_cases WHERE id LIKE 'turn_plan_%'\"",
        "curl -sf http://127.0.0.1:3400/health && echo",
    ]
    for cmd in cmds:
        print(f"\n>>> {cmd}\n", flush=True)
        _, stdout, stderr = client.exec_command(cmd, timeout=120)
        out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
        print(out.strip())
        if stdout.channel.recv_exit_status() != 0 and "health" not in cmd:
            client.close()
            return 1
    client.close()
    print("\nTurnPlan eval DB seed OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
