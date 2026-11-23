#!/usr/bin/env python3
import json
import sys
from pathlib import Path

import paramiko

SKILL = Path.home() / ".cursor" / "skills" / "remote-deploy" / "scripts"
sys.path.insert(0, str(SKILL))
from credentials import resolve_ssh_cfg  # noqa: E402

DEPLOY = Path.home() / ".cursor" / "skills" / "remote-deploy" / "deploy.json"
path = sys.argv[1] if len(sys.argv) > 1 else "/macro/jev/results/symbols?date=2026-11-23&pool_workflow=cn_a&source=canonical"


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
    safe_path = path.replace('"', "").replace("&", "\\&")
    cmd = (
        "set -a && . /home/ubuntu/.geegoo/agent.env && set +a && "
        f'curl -sS -H "Authorization: Bearer $MACRO_API_KEY" '
        f'"http://146.56.225.252:3220{safe_path}"'
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=90)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    client.close()
    if err.strip():
        print(err, file=sys.stderr)
    print(out[:8000])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())