#!/usr/bin/env python3
"""Check Futu / trade stack on GeeGooBot host."""
from __future__ import annotations

import json
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


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    print("=== processes (mcp / bot / futu) ===")
    print(
        ssh_run(
            bot_ssh,
            "ps aux | grep -E 'mcp-api|botAPIServer|FutuOpenD|mcpAPIServer' | grep -v grep",
        )
    )
    print("\n=== listening ports 3120/7000/11111 ===")
    print(ssh_run(bot_ssh, "ss -lntp 2>/dev/null | grep -E ':3120|:7000|:11111' || true"))
    print("\n=== python listen ports ===")
    print(ssh_run(bot_ssh, "ss -lntp 2>/dev/null | grep python || true"))
    print("\n=== TradingBot Bot_Server config ===")
    print(
        ssh_run(
            bot_ssh,
            "grep -E 'Bot_Server|bot_port|7000|5700' /home/ubuntu/apps/TradingBot/Config/APIConnection.py 2>/dev/null | head -8",
        )
    )


if __name__ == "__main__":
    main()
