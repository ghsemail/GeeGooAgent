#!/usr/bin/env python3
"""Patch GEEGOO_SIGNAL_CORS_ORIGINS for localhost dev + restart catalog-api."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

NEW_CORS = (
    "http://146.56.225.252:8088,"
    "http://146.56.225.252,"
    "http://localhost,"
    "http://127.0.0.1,"
    "http://localhost:8080,"
    "http://127.0.0.1:8080,"
    "http://localhost:5000,"
    "http://127.0.0.1:5000,"
    "http://localhost:3000,"
    "http://127.0.0.1:3000"
)


def run(client: paramiko.SSHClient, cmd: str) -> tuple[str, str]:
    _, stdout, stderr = client.exec_command(cmd)
    return stdout.read().decode("utf-8", "replace"), stderr.read().decode("utf-8", "replace")


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    ssh_cfg = cfg["targets"]["geegoo-tradingsignal"]["ssh"]
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(
        ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=20,
    )

    env_path = "/root/apps/GeeGooSignal/.env"
    print("=== before ===")
    out, _ = run(ssh, f"grep GEEGOO_SIGNAL_CORS_ORIGINS {env_path} || true")
    print(out.strip())

    patch_cmd = (
        f"if grep -q '^GEEGOO_SIGNAL_CORS_ORIGINS=' {env_path}; then "
        f"sed -i 's|^GEEGOO_SIGNAL_CORS_ORIGINS=.*|GEEGOO_SIGNAL_CORS_ORIGINS={NEW_CORS}|' {env_path}; "
        f"else echo 'GEEGOO_SIGNAL_CORS_ORIGINS={NEW_CORS}' >> {env_path}; fi"
    )
    run(ssh, patch_cmd)

    print("\n=== after ===")
    out, _ = run(ssh, f"grep GEEGOO_SIGNAL_CORS_ORIGINS {env_path}")
    print(out.strip())

    print("\n=== restart catalog-api ===")
    out, err = run(ssh, "cd /root/apps/GeeGooSignal && printf '2\\n0\\n' | bash start.sh")
    print(out[-2000:] if len(out) > 2000 else out)
    if err.strip():
        print("STDERR:", err[-1000:])

    print("\n=== verify CORS localhost ===")
    out, _ = run(
        ssh,
        "curl -sI -X OPTIONS http://127.0.0.1:3210/login "
        "-H 'Origin: http://localhost:54321' "
        "-H 'Access-Control-Request-Method: POST'",
    )
    print(out.strip())

    print("\n=== verify login via op_catalog ===")
    out, _ = run(
        ssh,
        "curl -s -o /dev/null -w 'http_code=%{http_code}' "
        "-X POST http://127.0.0.1:8088/op_catalog/login "
        "-H 'Content-Type: application/json' "
        "-H 'Authorization: Bearer 850367bc6d5fe8a4a53f267f5c308ac6d2ec1474d1764fe9' "
        "-d '{\"username\":\"test\",\"password\":\"test\"}'",
    )
    print(out.strip())

    ssh.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
