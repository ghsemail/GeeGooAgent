#!/usr/bin/env python3
"""Run kline_pattern_validate on geegoo-agent with SIGNAL_API_KEY from GeeGooSignal .env."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY = Path.home() / ".cursor/skills/remote-deploy/deploy.json"
if not DEPLOY.is_file():
    DEPLOY = Path(__file__).resolve().parents[2] / ".cursor/skills/remote-deploy/deploy.json"


def main() -> int:
    sys.path.insert(0, str(DEPLOY.parent / "scripts"))
    from credentials import resolve_ssh_cfg  # noqa: E402

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8-sig"))
    agent = resolve_ssh_cfg(cfg["targets"]["geegoo-agent"]["ssh"], "geegoo-agent")
    sig = resolve_ssh_cfg(cfg["targets"]["geegoo-signal"]["ssh"], "geegoo-signal")
    if not agent.get("password") or not sig.get("password"):
        print("SSH password missing in env/credentials", file=sys.stderr)
        return 2

    cs = paramiko.SSHClient()
    cs.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    cs.connect(sig["host"], username=sig["user"], password=sig["password"], timeout=30)
    _, o, _ = cs.exec_command(
        "grep -E '^GEEGOO_SIGNAL_SIGNAL_API_KEY=' /root/apps/GeeGooSignal/.env | head -1",
        timeout=30,
    )
    line = o.read().decode("utf-8", errors="replace").strip()
    cs.close()
    key = line.split("=", 1)[1].strip().strip('"').strip("'") if "=" in line else ""

    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(agent["host"], username=agent["user"], password=agent["password"], timeout=30)
    remote = f"""
set -e
cd /home/ubuntu/.geegoo/geegoo-agent
git fetch origin cursor/kline-pattern-display-d1b0 && git checkout cursor/kline-pattern-display-d1b0
export SIGNAL_API_URL=http://146.56.225.252:3200
export SIGNAL_API_KEY={key}
go run ./scripts/eval/kline_pattern_validate/
"""
    _, o, e = c.exec_command(remote, timeout=600)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    c.close()
    print(out, end="" if out.endswith("\n") else "\n")
    if err:
        print(err, file=sys.stderr, end="" if err.endswith("\n") else "\n")
    return 0 if out else 1


if __name__ == "__main__":
    raise SystemExit(main())
