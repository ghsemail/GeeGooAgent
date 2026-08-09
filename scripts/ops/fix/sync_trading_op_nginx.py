#!/usr/bin/env python3
"""Sync trading_operation web into nginx container + show CORS env."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(client: paramiko.SSHClient, cmd: str) -> str:
    _, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode("utf-8", "replace")
    err = stderr.read().decode("utf-8", "replace")
    if out.strip():
        print(out.rstrip())
    if err.strip():
        print("STDERR:", err.rstrip())
    return out


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    t = cfg["targets"]["trading-operation"]
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(
        t["ssh"]["host"],
        port=int(t["ssh"].get("port", 22)),
        username=t["ssh"]["user"],
        password=t["ssh"].get("password"),
        timeout=20,
    )

    print("=== sync web -> nginx container ===")
    run(
        ssh,
        "NGINX=$(docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'); "
        "echo nginx=$NGINX; "
        "docker cp /root/apps/trading_operation/web/. ${NGINX}:/usr/share/nginx/html/; "
        "docker exec $NGINX ls -la /usr/share/nginx/html/main.dart.js",
    )

    print("\n=== GeeGooSignal env CORS ===")
    run(
        ssh,
        "grep -i cors /root/apps/GeeGooSignal/.env 2>/dev/null || "
        "grep -i cors /root/apps/GeeGooSignal/start.sh 2>/dev/null || true",
    )

    print("\n=== catalog-api process env CORS ===")
    run(
        ssh,
        "PID=$(ss -lntp | awk '/3210/{print $NF}' | sed -n 's/.*pid=\\([0-9]*\\).*/\\1/p' | head -1); "
        "echo pid=$PID; "
        "tr '\\0' '\\n' < /proc/$PID/environ 2>/dev/null | grep -i cors || true",
    )

    ssh.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
