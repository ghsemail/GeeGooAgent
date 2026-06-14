#!/usr/bin/env python3
"""Run ``geegoo update`` on the remote server via SSH."""

from __future__ import annotations

import os
import sys

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")
REMOTE_HOME = os.environ.get("GEEGOO_REMOTE_HOME", "/home/ubuntu/.geegoo")


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    cmd = (
        "source $HOME/.bashrc 2>/dev/null || true; "
        f"export GEEGOO_HOME={REMOTE_HOME}; "
        f"export PATH={REMOTE_HOME}/bin:$PATH; "
        f"export GEEGOO_CONFIG={REMOTE_HOME}/config.json; "
        f"export GEEGOO_INSTALL_DIR={REMOTE_HOME}/geegoo-agent; "
        f"if [ -f {REMOTE_HOME}/github_token ]; then "
        f"export GEEGOO_GITHUB_TOKEN=$(cat {REMOTE_HOME}/github_token); fi; "
        "if [ -f $HOME/.geegoo/env ]; then set -a; source $HOME/.geegoo/env; set +a; fi; "
        '[ -n "$GEEGOO_GITHUB_TOKEN" ] && echo GEEGOO_GITHUB_TOKEN=set || echo GEEGOO_GITHUB_TOKEN=not_set; '
        "geegoo update; "
        "echo ---; "
        "geegoo doctor"
    )
    print(f"==> geegoo update on {USER}@{HOST}")
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=900)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    if out.strip():
        print(out)
    if err.strip():
        print(err, file=sys.stderr)
    client.close()
    return code


if __name__ == "__main__":
    raise SystemExit(main())
