#!/usr/bin/env python3
"""Deep probe PG memory tables on agent server."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE = r"""
import json, subprocess

cfg = json.load(open('/home/ubuntu/.geegoo/config.json'))
dsn = (cfg.get('postgres_dsn') or cfg.get('pg_dsn') or '').strip()
print('config postgres dsn:', 'set' if dsn else 'unset')

if dsn:
    queries = [
        ("chunks total", "SELECT COUNT(*) FROM agent_memory_chunks"),
        ("chunks with embedding", "SELECT COUNT(*) FROM agent_memory_chunks WHERE embedding IS NOT NULL"),
        ("chunks by source", "SELECT source, COUNT(*) FROM agent_memory_chunks GROUP BY source ORDER BY 1"),
        ("episodes total", "SELECT COUNT(*) FROM agent_episodes"),
        ("episodes sample", "SELECT id, left(summary,120), user_id, happened_at FROM agent_episodes ORDER BY id DESC LIMIT 5"),
    ]
    for label, sql in queries:
        p = subprocess.run(['psql', dsn, '-c', sql], capture_output=True, text=True)
        print('\n===', label, '===')
        print(p.stdout or p.stderr)

# consolidation log hints
p = subprocess.run(['grep', '-i', 'consolidat', '/home/ubuntu/.geegoo/geegoo-agent/agent-runtime.out'], capture_output=True, text=True)
lines = (p.stdout or '').strip().splitlines()
print('\n=== consolidation log tail ===')
for ln in lines[-8:]:
    print(ln)
"""


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(f"python3 <<'PY'\n{REMOTE}\nPY", timeout=90)
    print(o.read().decode())
    err = e.read().decode()
    if err.strip():
        print(err)
    c.close()


if __name__ == "__main__":
    main()
