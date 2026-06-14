#!/usr/bin/env python3
"""Upload GeeGooAgent tarball to server and install (~/.geegoo)."""

from __future__ import annotations

import os
import sys
import tarfile
import tempfile
from pathlib import Path

import paramiko

HOST = os.environ.get("SSH_HOST", "119.45.16.112")
USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")
PROJECT = Path(__file__).resolve().parents[1]
REMOTE_HOME = os.environ.get("GEEGOO_REMOTE_HOME", "/home/ubuntu/.geegoo")
REMOTE_DIR = f"{REMOTE_HOME}/geegoo-agent"


def connect() -> paramiko.SSHClient:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(HOST, username=USER, password=PASS, timeout=25)
    return client


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 600) -> tuple[str, str, int]:
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    code = stdout.channel.recv_exit_status()
    return (
        stdout.read().decode("utf-8", errors="replace"),
        stderr.read().decode("utf-8", errors="replace"),
        code,
    )


def make_tarball() -> Path:
    skip_dirs = {".git", ".venv", "venv", "__pycache__", ".pytest_cache", "data", ".ruff_cache"}
    skip_names = {".env", "config.json", "config.local.json", "_server_config.json"}
    tmp = tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False)
    tmp_path = Path(tmp.name)
    tmp.close()

    def filter_tar(tarinfo: tarfile.TarInfo) -> tarfile.TarInfo | None:
        parts = Path(tarinfo.name).parts
        if any(part in skip_dirs for part in parts):
            return None
        if Path(tarinfo.name).name in skip_names:
            return None
        return tarinfo

    with tarfile.open(tmp_path, "w:gz") as tar:
        tar.add(PROJECT, arcname="geegoo-agent", filter=filter_tar)
    return tmp_path


def main() -> int:
    if not PASS:
        print("Set SSH_PASS", file=sys.stderr)
        return 1

    print(f"==> packaging {PROJECT}")
    tarball = make_tarball()
    remote_tar = "/tmp/geegoo-agent.tar.gz"
    print(f"==> tarball {tarball.stat().st_size // 1024} KB")

    client = connect()
    try:
        print(f"==> uploading to {remote_tar}")
        sftp = client.open_sftp()
        sftp.put(str(tarball), remote_tar)
        sftp.close()
        tarball.unlink(missing_ok=True)

        install_script = f"""
set -euo pipefail
GEEGOO_HOME="{REMOTE_HOME}"
INSTALL_DIR="{REMOTE_DIR}"
mkdir -p "$GEEGOO_HOME" "$GEEGOO_HOME/data" "$GEEGOO_HOME/bin"
rm -rf "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
tar -xzf {remote_tar} -C "$(dirname "$INSTALL_DIR")"
rm -f {remote_tar}
python3 -m venv "$INSTALL_DIR/venv"
source "$INSTALL_DIR/venv/bin/activate"
pip install -U pip wheel -q
pip install -e "$INSTALL_DIR[dev]" -q
ln -sf "$INSTALL_DIR/venv/bin/geegoo" "$GEEGOO_HOME/bin/geegoo"
ln -sf "$INSTALL_DIR/venv/bin/geegoo-agent" "$GEEGOO_HOME/bin/geegoo-agent" 2>/dev/null || true
CONFIG="$GEEGOO_HOME/config.json"
if [ ! -f "$CONFIG" ]; then
  cp "$INSTALL_DIR/config.example.json" "$CONFIG"
  chmod 600 "$CONFIG"
  python3 -c "import json; from pathlib import Path; p=Path('$CONFIG'); raw=json.loads(p.read_text()); raw['output_dir']='$GEEGOO_HOME/data'; p.write_text(json.dumps(raw, indent=2, ensure_ascii=False)+chr(10))"
fi
echo PATH_ADD="$GEEGOO_HOME/bin"
ls -la "$GEEGOO_HOME"
"$INSTALL_DIR/venv/bin/geegoo" --help | head -6
"""
        print("==> installing on server")
        out, err, code = run(client, install_script, timeout=900)
        if out.strip():
            print(out)
        if err.strip():
            print(err, file=sys.stderr)
        if code != 0:
            return code

        path_line = f'export PATH="{REMOTE_HOME}/bin:$PATH"'
        env_cmds = [
            f'grep -qF "GEEGOO_HOME=" ~/.bashrc 2>/dev/null || echo \'export GEEGOO_HOME="{REMOTE_HOME}"\' >> ~/.bashrc',
            f'grep -qF "GEEGOO_CONFIG=" ~/.bashrc 2>/dev/null || echo \'export GEEGOO_CONFIG="{REMOTE_HOME}/config.json"\' >> ~/.bashrc',
            f'grep -qF "{REMOTE_HOME}/bin" ~/.bashrc 2>/dev/null || echo \'{path_line}\' >> ~/.bashrc',
        ]
        for cmd in env_cmds:
            run(client, cmd, timeout=30)

        print("\n==> done")
        print(f"  repo:   {REMOTE_DIR}")
        print(f"  config: {REMOTE_HOME}/config.json")
        print("  next:   geegoo setup && geegoo doctor")
        return 0
    finally:
        client.close()


if __name__ == "__main__":
    raise SystemExit(main())
