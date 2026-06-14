#!/usr/bin/env python3
"""Patch remote ~/.geegoo/config.json and run geegoo doctor."""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")
REMOTE_CONFIG = os.environ.get("GEEGOO_REMOTE_CONFIG", "/home/ubuntu/.geegoo/config.json")
GEEGOO_SKILL_CONFIG = Path(
    os.environ.get("GEEGOO_SKILL_CONFIG", r"C:\Users\ghsemail\.cursor\skills\geegoo\config.json")
)


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    geegoo_cfg = json.loads(GEEGOO_SKILL_CONFIG.read_text(encoding="utf-8"))
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    sftp = client.open_sftp()
    with sftp.open(REMOTE_CONFIG, "r") as remote_file:
        raw = json.loads(remote_file.read().decode("utf-8"))

    raw["base_url"] = geegoo_cfg["base_url"]
    raw["geegoo_url"] = geegoo_cfg["base_url"]
    raw["api_key"] = geegoo_cfg["api_key"]
    raw["geegoo_api_key"] = geegoo_cfg["api_key"]
    raw["mcp_token"] = geegoo_cfg["mcp_token"]
    raw["output_dir"] = "/home/ubuntu/.geegoo/data"

    with sftp.open(REMOTE_CONFIG, "w") as remote_file:
        remote_file.write(json.dumps(raw, indent=2, ensure_ascii=False) + "\n")
    sftp.close()
    print("remote config updated (api + mcp + urls)")

    def run(cmd: str, timeout: int = 180) -> tuple[str, str, int]:
        _stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
        code = stdout.channel.recv_exit_status()
        return (
            stdout.read().decode("utf-8", errors="replace"),
            stderr.read().decode("utf-8", errors="replace"),
            code,
        )

    out, _, _ = run("test -f ~/.geegoo/env && wc -l ~/.geegoo/env || echo no_env_file")
    print(out.strip())

    env_prefix = (
        "export PATH=/home/ubuntu/.geegoo/bin:$PATH; "
        "export GEEGOO_HOME=/home/ubuntu/.geegoo; "
        "export GEEGOO_CONFIG=/home/ubuntu/.geegoo/config.json; "
    )
    out, err, code = run(f"{env_prefix} geegoo doctor 2>&1", timeout=120)
    print("--- geegoo doctor ---")
    print(out)
    if err.strip():
        print(err, file=sys.stderr)

    out, _, _ = run(f"{env_prefix} test -f ~/.geegoo/geegoo-agent/src/geegoo_agent/clients/geegoo_bot.py && echo geegoo_bot_client_ok")
    print(out.strip())

    client.close()
    return code


if __name__ == "__main__":
    raise SystemExit(main())
