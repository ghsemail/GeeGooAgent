#!/usr/bin/env python3
"""Deploy TurnPlan live eval branch to GeeGoo agent runtime + migrate eval DB."""
from __future__ import annotations

import os
import sys
from pathlib import Path

import paramiko

REPO = "/home/ubuntu/.geegoo/geegoo-agent"
BRANCH = os.environ.get("DEPLOY_BRANCH", "cursor/fix-turnplan-eval-live-5d18")
SQL_FILE = Path(__file__).resolve().parents[3] / "internal/infra/pgschema/postgres_eval.sql"


def ssh_run(client: paramiko.SSHClient, cmd: str, timeout: int = 900) -> tuple[int, str]:
    print(f"\n>>> {cmd}\n")
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    if out.strip():
        tail = out[-8000:] if len(out) > 8000 else out
        print(tail)
    return stdout.channel.recv_exit_status(), out


def main() -> int:
    password = os.environ.get("GEEGOO_AGENT_SSH_PASSWORD")
    host = os.environ.get("GEEGOO_AGENT_SSH_HOST")
    user = os.environ.get("GEEGOO_AGENT_SSH_USER", "ubuntu")
    if not password or not host:
        print("missing GEEGOO_AGENT_SSH_PASSWORD or GEEGOO_AGENT_SSH_HOST", file=sys.stderr)
        return 1

    sql_lines = SQL_FILE.read_text(encoding="utf-8").splitlines()
    migration = [ln for ln in sql_lines if ln.startswith("DELETE FROM agent_eval_cases WHERE id = 'turn_plan_routing'") or ln.startswith("INSERT INTO agent_eval_cases") and "turn_plan_" in ln]
    if len(migration) < 22:
        print(f"expected >=22 migration lines, got {len(migration)}", file=sys.stderr)
        return 1
    migration_sql = "\n".join(migration) + "\n"

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host, username=user, password=password, timeout=30)
    sftp = client.open_sftp()
    remote_sql = "/tmp/turnplan_live_eval.sql"
    with sftp.file(remote_sql, "w") as f:
        f.write(migration_sql)
    sftp.close()

    steps = [
        f"cd {REPO} && git fetch origin {BRANCH} && git checkout {BRANCH} && git reset --hard origin/{BRANCH} && git log -1 --oneline",
        f"cd {REPO} && bash start.sh build",
        f"cd {REPO} && bash start.sh restart-runtime",
        "sleep 3",
        "curl -sf http://127.0.0.1:3400/health",
        f"set -a && source /home/ubuntu/.geegoo/agent.env && set +a && psql \"$GEEGOO_PG_DSN\" -v ON_ERROR_STOP=1 -f {remote_sql}",
        f"set -a && source /home/ubuntu/.geegoo/agent.env && set +a && psql \"$GEEGOO_PG_DSN\" -tAc \"SELECT count(*) FROM agent_eval_cases WHERE id LIKE 'turn_plan_%'\"",
        "curl -s -o /dev/null -w 'verify:%{http_code}\\n' -H 'Authorization: Bearer test-runtime-key' -X POST http://127.0.0.1:3400/v1/dashboard/eval/cases/turn_plan_stock_price_lookup/verify",
        "curl -s -o /dev/null -w 'run:%{http_code}\\n' -H 'Authorization: Bearer test-runtime-key' -X POST http://127.0.0.1:3400/v1/dashboard/eval/cases/turn_plan_stock_price_lookup/run",
        "curl -s -H 'Authorization: Bearer test-runtime-key' http://127.0.0.1:3400/v1/dashboard/eval/cases | python3 -c \"import sys,json;d=json.load(sys.stdin);ids=[c['id'] for c in d.get('cases',[]) if c['id'].startswith('turn_plan_')];print('turn_plan_cases',len(ids));print(ids[:3], '...', ids[-1 if ids else 0])\"",
    ]
    try:
        for cmd in steps:
            code, out = ssh_run(client, cmd)
            if code != 0 and "curl -s -o" not in cmd:
                print(f"FAILED exit {code}")
                return code
            if "verify:404" in out or "run:404" in out:
                print("route still missing after deploy")
                return 1
    finally:
        client.close()
    print("\n=== TurnPlan live eval deploy OK ===")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
