#!/usr/bin/env python3
"""Ensure ~/.geegoo/bin is on PATH in ~/.bashrc and verify geegoo command."""

from __future__ import annotations

import os
import sys

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    def run(cmd: str) -> tuple[str, str, int]:
        _stdin, stdout, stderr = client.exec_command(cmd, timeout=60)
        code = stdout.channel.recv_exit_status()
        return (
            stdout.read().decode("utf-8", errors="replace"),
            stderr.read().decode("utf-8", errors="replace"),
            code,
        )

    fix = r"""
GEEGOO_HOME="$HOME/.geegoo"
grep -qF '.geegoo/bin' ~/.bashrc 2>/dev/null || echo 'export PATH="$HOME/.geegoo/bin:$PATH"' >> ~/.bashrc
grep -qF 'GEEGOO_HOME=' ~/.bashrc 2>/dev/null || echo 'export GEEGOO_HOME="$HOME/.geegoo"' >> ~/.bashrc
grep -qF 'GEEGOO_CONFIG=' ~/.bashrc 2>/dev/null || echo 'export GEEGOO_CONFIG="$HOME/.geegoo/config.json"' >> ~/.bashrc
grep -qF '.geegoo/bin' ~/.profile 2>/dev/null || {
  echo '' >> ~/.profile
  echo '# GeeGoo Agent' >> ~/.profile
  echo 'export GEEGOO_HOME="$HOME/.geegoo"' >> ~/.profile
  echo 'export GEEGOO_CONFIG="$HOME/.geegoo/config.json"' >> ~/.profile
  echo 'export PATH="$HOME/.geegoo/bin:$PATH"' >> ~/.profile
}
echo 'fixed profile + bashrc'
grep -E 'GEEGOO|\.geegoo/bin' ~/.profile ~/.bashrc 2>/dev/null || true
bash -lc 'echo PATH=$PATH; command -v geegoo; geegoo --help | head -4'
"""
    out, err, code = run(fix)
    print(out)
    if err.strip():
        print(err, file=sys.stderr)
    client.close()
    return code


if __name__ == "__main__":
    raise SystemExit(main())
