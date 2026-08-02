#!/usr/bin/env python3
"""Diagnose trading_operation login (/op_catalog) on signal host."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(client: paramiko.SSHClient, cmd: str) -> tuple[str, str]:
    _, stdout, stderr = client.exec_command(cmd)
    return stdout.read().decode("utf-8", "replace"), stderr.read().decode("utf-8", "replace")


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

    checks = [
        ("docker ps (nginx/8088)", "docker ps --format 'table {{.Names}}\t{{.Ports}}'"),
        ("curl 8088 root", "curl -sI http://127.0.0.1:8088/ | head -5"),
        (
            "curl op_catalog login via nginx",
            "curl -s -o /tmp/op_login.out -w 'http_code=%{http_code}' "
            "-X POST http://127.0.0.1:8088/op_catalog/login "
            "-H 'Content-Type: application/json' "
            "-d '{\"username\":\"test\",\"password\":\"test\"}' ; echo ; head -c 500 /tmp/op_login.out ; echo",
        ),
        (
            "curl catalog-api direct 3210",
            "curl -s -o /tmp/cat_login.out -w 'http_code=%{http_code}' "
            "-X POST http://127.0.0.1:3210/login "
            "-H 'Content-Type: application/json' "
            "-d '{\"username\":\"test\",\"password\":\"test\"}' ; echo ; head -c 500 /tmp/cat_login.out ; echo",
        ),
        ("listen 3210", "ss -lntp | grep 3210 || true"),
        ("listen 3200", "ss -lntp | grep 3200 || true"),
        (
            "nginx op_catalog config",
            "CID=$(docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'); "
            "echo container=$CID; "
            "docker exec $CID sh -c 'grep -R op_catalog /etc/nginx 2>/dev/null; grep -R op_signal /etc/nginx 2>/dev/null' || true",
        ),
        (
            "catalog-api process",
            "ps aux | grep -E 'catalog|3210' | grep -v grep | head -5 || true",
        ),
        (
            "CORS preflight catalog 3210 localhost",
            "curl -sI -X OPTIONS http://127.0.0.1:3210/login "
            "-H 'Origin: http://localhost:12345' "
            "-H 'Access-Control-Request-Method: POST'",
        ),
        (
            "CORS preflight via nginx op_catalog",
            "curl -sI -X OPTIONS http://127.0.0.1:8088/op_catalog/login "
            "-H 'Origin: http://146.56.225.252:8088' "
            "-H 'Access-Control-Request-Method: POST'",
        ),
        (
            "nginx default.conf",
            "docker exec 0cb244428c30 cat /etc/nginx/conf.d/default.conf",
        ),
        (
            "deployed web has op_catalog",
            "grep -c op_catalog /root/apps/trading_operation/web/main.dart.js 2>/dev/null; "
            "ls -la /root/apps/trading_operation/web/main.dart.js 2>/dev/null; "
            "docker exec 0cb244428c30 sh -c 'grep -c op_catalog /usr/share/nginx/html/main.dart.js; ls -la /usr/share/nginx/html/main.dart.js'",
        ),
    ]

    for title, cmd in checks:
        print(f"\n=== {title} ===")
        out, err = run(ssh, cmd)
        if out.strip():
            print(out.rstrip())
        if err.strip():
            print("STDERR:", err.rstrip())

    ssh.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
