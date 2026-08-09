#!/usr/bin/env python3
"""Ensure local MongoDB docker on GeeGooData HK/CN nodes and restart if crash-looping."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

ENSURE_MONGO = r"""
set -e
DOCKER="docker"
if ! docker ps >/dev/null 2>&1; then DOCKER="sudo docker"; fi
if $DOCKER ps -a --format '{{.Names}} {{.Status}}' 2>/dev/null | grep -q '^mongodb '; then
  if ! $DOCKER ps --format '{{.Names}}' | grep -qx mongodb; then
    $DOCKER rm -f mongodb 2>/dev/null || true
  fi
fi
if ! $DOCKER ps --format '{{.Names}}' 2>/dev/null | grep -qx mongodb; then
  mkdir -p /data/mongodb
  $DOCKER run -d --name mongodb --restart unless-stopped \
    -p 127.0.0.1:27017:27017 -v /data/mongodb:/data/db \
    mongo:7 --wiredTigerCacheSizeGB 0.25
fi
sleep 3
$DOCKER ps --format '{{.Names}} {{.Status}}' | grep mongo || echo NO_MONGO
ss -lntp | grep 27017 || echo NO_27017
"""


def run(target: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh.get("password"), timeout=60)
    _, o, e = c.exec_command(ENSURE_MONGO, timeout=120)
    out = (o.read() + e.read()).decode()
    c.close()
    return out


def main() -> int:
    for target in ("geegoo-data", "geegoo-data-cn"):
        print(f"## {target}")
        print(run(target))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
