#!/usr/bin/env python3
"""Write GitHub token to remote ~/.geegoo/github_token for private-repo ``geegoo update``."""

from __future__ import annotations

import os
import sys

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")
REMOTE_HOME = os.environ.get("GEEGOO_REMOTE_HOME", "/home/ubuntu/.geegoo")
TOKEN = os.environ.get("GEEGOO_GITHUB_TOKEN", "").strip()


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1
    if not TOKEN:
        print(
            "Set GEEGOO_GITHUB_TOKEN (fine-grained or classic PAT with repo read).",
            file=sys.stderr,
        )
        return 1
    if any(ch in TOKEN for ch in ("\n", "\r")):
        print("Token must be a single line.", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)
    try:
        sftp = client.open_sftp()
        remote_dir = REMOTE_HOME
        try:
            sftp.stat(remote_dir)
        except FileNotFoundError:
            sftp.mkdir(remote_dir)
        remote_path = f"{remote_dir}/github_token"
        with sftp.file(remote_path, "w") as handle:
            handle.write(TOKEN)
        sftp.chmod(remote_path, 0o600)
        sftp.close()
        print(f"==> wrote {remote_path} (chmod 600) on {USER}@{HOST}")

        cmd = (
            f"export GEEGOO_GITHUB_TOKEN=$(cat {remote_path}); "
            f"export PATH={REMOTE_HOME}/bin:$PATH; "
            f"export GEEGOO_INSTALL_DIR={REMOTE_HOME}/geegoo-agent; "
            "geegoo update; echo ---; geegoo doctor"
        )
        _stdin, stdout, stderr = client.exec_command(cmd, timeout=900)
        out = stdout.read().decode("utf-8", errors="replace")
        err = stderr.read().decode("utf-8", errors="replace")
        code = stdout.channel.recv_exit_status()
        if out.strip():
            print(out)
        if err.strip():
            print(err, file=sys.stderr)
        return code
    finally:
        client.close()


if __name__ == "__main__":
    raise SystemExit(main())
