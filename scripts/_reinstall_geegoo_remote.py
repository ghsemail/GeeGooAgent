#!/usr/bin/env python3
"""Remove GigoAgent and reinstall GeeGooAgent on 119.45.16.112."""

from __future__ import annotations

import os
import sys

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "Ghs@2024")
INSTALL_URL = os.environ.get(
    "GEEGOO_INSTALL_URL",
    "https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh",
)


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 600) -> tuple[str, str, int]:
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    code = stdout.channel.recv_exit_status()
    return (
        stdout.read().decode("utf-8", errors="replace"),
        stderr.read().decode("utf-8", errors="replace"),
        code,
    )


def main() -> int:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    print(f"==> connecting {USER}@{HOST}")
    client.connect(HOST, username=USER, password=PASS, timeout=25)

    migrate_and_cleanup = r"""
set -e
BACKUP=/tmp/gigo-to-geegoo-backup
rm -rf "$BACKUP"
mkdir -p "$BACKUP"
if [ -d "$HOME/.gigo" ]; then
  cp -a "$HOME/.gigo/config.json" "$BACKUP/" 2>/dev/null || true
  cp -a "$HOME/.gigo/github_token" "$BACKUP/" 2>/dev/null || true
  cp -a "$HOME/.gigo/data" "$BACKUP/" 2>/dev/null || true
  echo "backed up gigo config/data to $BACKUP"
  rm -rf "$HOME/.gigo"
  echo "removed ~/.gigo"
else
  echo "no ~/.gigo found"
fi
# remove gigo from shell rc
for rc in "$HOME/.bashrc" "$HOME/.profile"; do
  if [ -f "$rc" ]; then
    sed -i '/GIGO_HOME=/d;/GIGO_CONFIG=/d;/\.gigo\/bin/d' "$rc" || true
  fi
done
echo DONE_CLEANUP
"""
    print("==> backup + remove GigoAgent")
    out, err, code = run(client, migrate_and_cleanup, timeout=120)
    print(out)
    if err.strip():
        print(err, file=sys.stderr)
    if code != 0:
        print(f"cleanup failed: {code}", file=sys.stderr)
        client.close()
        return code

    install_cmd = (
        f"export GEEGOO_SKIP_SETUP=1 DEBIAN_FRONTEND=noninteractive; "
        f"curl -fsSL {INSTALL_URL} | bash"
    )
    print("==> install GeeGooAgent")
    out, err, code = run(client, install_cmd, timeout=600)
    print(out)
    if err.strip():
        print(err, file=sys.stderr)

    restore = r"""
set -e
BACKUP=/tmp/gigo-to-geegoo-backup
GEEGOO_HOME="$HOME/.geegoo"
CONFIG="$GEEGOO_HOME/config.json"
if [ -f "$BACKUP/config.json" ] && [ -f "$CONFIG" ]; then
  python3 - <<'PY'
import json
from pathlib import Path
backup = Path("/tmp/gigo-to-geegoo-backup/config.json")
target = Path.home() / ".geegoo" / "config.json"
old = json.loads(backup.read_text(encoding="utf-8"))
new = json.loads(target.read_text(encoding="utf-8"))
for key in ("llm", "mcp_token", "mcp_base_url", "output_dir", "github_token_env"):
    if key in old and old[key]:
        new[key] = old[key]
if "output_dir" not in new or not new["output_dir"]:
    new["output_dir"] = str(Path.home() / ".geegoo" / "data")
target.write_text(json.dumps(new, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
print("merged config from gigo backup")
PY
fi
if [ -f "$BACKUP/github_token" ]; then
  cp -a "$BACKUP/github_token" "$GEEGOO_HOME/"
  chmod 600 "$GEEGOO_HOME/github_token"
  echo "restored github_token"
fi
if [ -d "$BACKUP/data" ] && [ ! "$(ls -A "$GEEGOO_HOME/data" 2>/dev/null)" ]; then
  cp -a "$BACKUP/data/." "$GEEGOO_HOME/data/"
  echo "restored data dir"
fi
rm -rf "$BACKUP"
echo DONE_RESTORE
"""
    print("==> restore config from backup")
    out, err, _ = run(client, restore, timeout=120)
    print(out)
    if err.strip():
        print(err, file=sys.stderr)

    verify_cmds = [
        "test ! -d ~/.gigo && echo 'gigo removed OK' || echo 'gigo still exists'",
        "ls -la ~/.geegoo/",
        "ls -la ~/.geegoo/geegoo-agent/ | head -10",
        "test -x ~/.geegoo/bin/geegoo && ~/.geegoo/bin/geegoo --help | head -5",
        "test -f ~/.geegoo/config.json && echo config_ok",
        "grep -E 'GEEGOO_HOME|GEEGOO_CONFIG|\.geegoo/bin' ~/.bashrc | tail -5 || true",
    ]
    print("==> verify")
    for cmd in verify_cmds:
        print(f"\n--- {cmd}")
        out, err, _ = run(client, cmd, timeout=30)
        print(out.strip() or err.strip())

    client.close()
    print(f"\n==> install exit code: {code}")
    return code


if __name__ == "__main__":
    raise SystemExit(main())
