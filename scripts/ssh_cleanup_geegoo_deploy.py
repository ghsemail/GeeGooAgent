#!/usr/bin/env python3
"""Remove GeeGoo Agent deployment from a remote server."""

from __future__ import annotations

import os
import sys

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 60) -> tuple[str, str]:
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    return (
        stdout.read().decode("utf-8", errors="replace"),
        stderr.read().decode("utf-8", errors="replace"),
    )


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    probe_cmds = [
        "whoami && hostname",
        "systemctl list-unit-files 'geegoo*' 2>/dev/null; systemctl list-units 'geegoo*' --all 2>/dev/null",
        "ls -la /opt/geegoo-agent 2>/dev/null || echo 'no /opt/geegoo-agent'",
        "ls -la /etc/geegoo-agent 2>/dev/null || echo 'no /etc/geegoo-agent'",
        "ls -la /var/lib/geegoo-agent 2>/dev/null || echo 'no /var/lib/geegoo-agent'",
        "id geegoo-agent 2>/dev/null || echo 'no geegoo-agent user'",
        "grep -r geegoo-agent /etc/systemd/system/ 2>/dev/null | head -20 || true",
    ]
    print("=== BEFORE ===")
    for cmd in probe_cmds:
        print(f"--- {cmd[:70]}")
        out, err = run(client, cmd)
        print(out.strip() or err.strip()[:500])

    cleanup = r"""
set -e
echo '=== STOP & DISABLE ==='
for unit in geegoo-agent-pre-market.timer geegoo-agent-pre-market.service geegoo-agent.service geegoo-agent.timer; do
  sudo systemctl stop "$unit" 2>/dev/null || true
  sudo systemctl disable "$unit" 2>/dev/null || true
done

echo '=== REMOVE SYSTEMD UNITS ==='
for f in /etc/systemd/system/geegoo-agent-pre-market.timer \
         /etc/systemd/system/geegoo-agent-pre-market.service \
         /etc/systemd/system/geegoo-agent.service \
         /etc/systemd/system/geegoo-agent.timer; do
  if [ -f "$f" ]; then sudo rm -f "$f"; echo removed "$f"; fi
done
sudo systemctl daemon-reload 2>/dev/null || true

echo '=== REMOVE DIRS ==='
for d in /opt/geegoo-agent /etc/geegoo-agent /var/lib/geegoo-agent; do
  if [ -d "$d" ]; then sudo rm -rf "$d"; echo removed "$d"; fi
done

echo '=== REMOVE USER (optional) ==='
if id geegoo-agent >/dev/null 2>&1; then
  sudo userdel -r geegoo-agent 2>/dev/null || sudo userdel geegoo-agent 2>/dev/null || echo 'userdel skipped'
fi

echo '=== VERIFY ==='
systemctl list-unit-files 'geegoo*' 2>/dev/null || echo 'no geegoo units'
ls /opt/geegoo-agent /etc/geegoo-agent /var/lib/geegoo-agent 2>&1 || true
id geegoo-agent 2>&1 || echo 'geegoo-agent user gone'
echo DONE
"""
    print("\n=== CLEANUP ===")
    out, err = run(client, cleanup, timeout=120)
    print(out)
    if err.strip():
        print("STDERR:", err[:800])

    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
