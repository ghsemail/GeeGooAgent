#!/usr/bin/env python3
"""Patch remote ~/.geegoo/config.json API Bearer keys (keep mcp_token + llm)."""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

import paramiko

from geegoo_agent.infra.tradingbot_sync import build_config

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")
TRADINGBOT = Path(os.environ.get("TRADINGBOT_PATH", r"D:\Geegoo\TradingBot"))
REMOTE_CONFIG = os.environ.get("GEEGOO_REMOTE_CONFIG", "/home/ubuntu/.geegoo/config.json")


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1
    if not TRADINGBOT.is_dir():
        print(f"TradingBot not found: {TRADINGBOT}", file=sys.stderr)
        return 1

    synced = build_config(TRADINGBOT.resolve())
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    sftp = client.open_sftp()
    with sftp.open(REMOTE_CONFIG, "r") as remote_file:
        raw = json.loads(remote_file.read().decode("utf-8"))

    for key in ("base_url", "api_key", "geegoo_url", "geegoo_api_key", "signal_base_url"):
        if key in synced:
            raw[key] = synced[key]
    sandbox = raw.setdefault("sandbox", {})
    if isinstance(sandbox, dict) and "allowed_hosts" in synced.get("sandbox", {}):
        sandbox["allowed_hosts"] = synced["sandbox"]["allowed_hosts"]

    payload = json.dumps(raw, indent=2, ensure_ascii=False) + "\n"
    with sftp.open(REMOTE_CONFIG, "w") as remote_file:
        remote_file.write(payload.encode("utf-8"))
    sftp.close()

    doctor_cmd = "/home/ubuntu/.geegoo/geegoo-agent/venv/bin/geegoo doctor --skip-llm"
    _stdin, stdout, stderr = client.exec_command(doctor_cmd, timeout=120)
    code = stdout.channel.recv_exit_status()
    print(stdout.read().decode("utf-8", errors="replace"))
    err = stderr.read().decode("utf-8", errors="replace")
    if err.strip():
        print(err, file=sys.stderr)
    client.close()
    return code


if __name__ == "__main__":
    sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))
    raise SystemExit(main())
