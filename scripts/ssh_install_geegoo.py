#!/usr/bin/env python3
"""Run install.sh on remote server (download only, no interactive setup)."""

from __future__ import annotations

import os
import sys

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")
INSTALL_URL = os.environ.get(
    "GEEGOO_INSTALL_URL",
    "https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh",
)


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 600) -> tuple[str, str, int]:
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    exit_code = stdout.channel.recv_exit_status()
    return (
        stdout.read().decode("utf-8", errors="replace"),
        stderr.read().decode("utf-8", errors="replace"),
        exit_code,
    )


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    print(f"==> connecting {USER}@{HOST}")
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    probe_cmds = [
        "whoami && hostname && uname -a",
        "python3 --version 2>&1; git --version 2>&1",
        "ls -la ~/.geegoo 2>/dev/null || echo 'no ~/.geegoo yet'",
    ]
    for cmd in probe_cmds:
        print(f"\n--- {cmd}")
        out, err, code = run(client, cmd, timeout=30)
        print(out.strip() or err.strip())

    install_cmd = (
        f"export GEEGOO_SKIP_SETUP=1 DEBIAN_FRONTEND=noninteractive; "
        f"curl -fsSL {INSTALL_URL} | bash"
    )
    print(f"\n==> running install.sh (download only)")
    out, err, code = run(client, install_cmd, timeout=600)
    print(out)
    if err.strip():
        print(err, file=sys.stderr)

    verify_cmds = [
        "ls -la ~/.geegoo/",
        "ls -la ~/.geegoo/geegoo-agent/ | head -15",
        "test -x ~/.geegoo/bin/geegoo && ~/.geegoo/bin/geegoo --help | head -5 || echo geegoo not linked",
        "test -f ~/.geegoo/config.json && echo config_exists || echo no_config",
    ]
    print("\n==> verify")
    for cmd in verify_cmds:
        print(f"\n--- {cmd}")
        out, err, _ = run(client, cmd, timeout=30)
        print(out.strip() or err.strip())

    client.close()
    print(f"\n==> install exit code: {code}")
    return code


if __name__ == "__main__":
    raise SystemExit(main())
