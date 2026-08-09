#!/usr/bin/env python3
"""Compare Go mcp :3120 vs legacy Python mcp :5700 for Futu tools."""
from __future__ import annotations

import json
import re
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(ssh_cfg: dict, cmd: str) -> str:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=60)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    client.close()
    return (out + err).strip()


def curl_port(port: int, path: str, api_key: str, body: dict) -> str:
    payload = json.dumps(body, ensure_ascii=False)
    return (
        f"curl -sS -m 45 -w '\\nHTTP:%{{http_code}}' "
        f"-H 'Authorization: Bearer {api_key}' -H 'Content-Type: application/json' "
        f"-d '{payload}' http://127.0.0.1:{port}/{path}"
    )


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent_ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    out = ssh_run(
        agent_ssh,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token','')); print(c.get('geegoo_api_key',''))\"",
    )
    lines = [ln.strip() for ln in out.splitlines() if ln.strip()]
    token, api_key = lines[0], lines[1]

    body = {"mcp_token": token, "code": "00700.HK", "num": 5}
    for port, label in [(3120, "Go mcp-api"), (5700, "Python mcp-api")]:
        print(f"\n=== {label} :{port} /getTicker ===")
        raw = ssh_run(bot_ssh, curl_port(port, "getTicker", api_key, body))
        print(raw[:600])


if __name__ == "__main__":
    main()
