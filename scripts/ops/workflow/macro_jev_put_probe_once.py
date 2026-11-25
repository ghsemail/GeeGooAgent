#!/usr/bin/env python3
"""模拟重复 PUT 以读取 Signal 返回的 reason（在 Agent 机执行 curl）。"""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
BODY = json.dumps(
    {
        "date": "2026-11-23",
        "pool_workflow": "cn_a",
        "symbol": "000001.SH",
        "status": "success",
        "jev_direction": "short",
        "jev_reason": "probe repeat put to read stale reason",
        "confidence": 0.75,
        "detail_ref": "diagnose_macro",
        "agent_session_id": "probe",
        "agent_run_id": "probe",
        "source_batch_id": "macro-jev-cn_a-2026-11-23",
        "batch_started_at": "2026-11-23T15:50:05Z",
        "completed_at": "2026-11-23T15:54:00Z",
        "revision": 2,
    },
    ensure_ascii=False,
)


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh_cfg = resolve_ssh_cfg(cfg["targets"]["geegoo-agent"]["ssh"], "geegoo-agent")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        key_filename=ssh_cfg.get("key_file"),
        timeout=30,
    )
    import base64

    b64 = base64.b64encode(BODY.encode("utf-8")).decode("ascii")
    cmd = (
        "set -a && . /home/ubuntu/.geegoo/agent.env && set +a && "
        f'echo {b64} | base64 -d | curl -sS -X PUT -H "Authorization: Bearer $MACRO_API_KEY" '
        '-H "Content-Type: application/json" -d @- '
        '"http://146.56.225.252:3220/macro/jev/results/symbols"'
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=60)
    print(stdout.read().decode("utf-8", errors="replace"))
    err = stderr.read().decode("utf-8", errors="replace")
    if err.strip():
        print(err, file=sys.stderr)
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())