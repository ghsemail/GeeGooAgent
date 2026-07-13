#!/usr/bin/env python3
"""Quick verify getCurrentPrice on GeeGooBot mcp-api."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 90) -> tuple[int, str, str]:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    return code, out, err


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent_ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]

    code, out, _ = ssh_run(
        agent_ssh,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token','')); print(c.get('geegoo_api_key',''))\"",
    )
    lines = [ln.strip() for ln in out.strip().splitlines() if ln.strip()]
    token, api_key = lines[0], lines[1]

    for endpoint, body in [
        ("getSinglePromptTemplate", {"mcp_token": token, "type": "tech", "period": "daily"}),
        ("getCurrentPrice", {"mcp_token": token, "code": "00700.HK"}),
    ]:
        payload = json.dumps(body)
        cmd = (
            f"curl -sS -m 60 -w '\\nHTTP:%{{http_code}}' "
            f"-H 'Authorization: Bearer {api_key}' -H 'Content-Type: application/json' "
            f"-d '{payload}' http://127.0.0.1:3120/{endpoint}"
        )
        code, out, err = ssh_run(bot_ssh, cmd, timeout=90)
        print(f"\n=== POST /{endpoint} (exit {code}) ===")
        print(out[-600:])
        if err.strip():
            print("stderr:", err[:200])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
