#!/usr/bin/env python3
"""Patch GeeGooAgent production config → GeeGooBot mcp-api :3120 and verify."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
REMOTE_CONFIG = "/home/ubuntu/.geegoo/config.json"
GEEGOO_URL = "http://118.195.135.97:3120"
SIGNAL_URL = "http://146.56.225.252:3210"


def fetch_mcp_api_key(bot_client: paramiko.SSHClient) -> str:
    _, o, _ = bot_client.exec_command(
        "python3 -c \"import re; t=open('/home/ubuntu/apps/TradingBot/mcp/constants.py').read(); "
        "m=re.search(r'API_KEY\\s*=\\s*\\\"([^\\\"]+)\\\"', t); print(m.group(1) if m else '')\"",
        timeout=30,
    )
    key = o.read().decode().strip()
    if not key:
        local = Path(r"D:\Geegoo\TradingBot\mcp\constants.py")
        if local.is_file():
            m = re.search(r'API_KEY\s*=\s*["\']([^"\']+)["\']', local.read_text(encoding="utf-8"))
            key = m.group(1) if m else ""
    if not key:
        raise RuntimeError("cannot resolve MCP API_KEY")
    return key


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 300) -> tuple[int, str]:
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    return stdout.channel.recv_exit_status(), out


def main() -> int:
    cfg = json.load(DEPLOY_CFG.open(encoding="utf-8"))
    agent_ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    bot_ssh = cfg["targets"]["geegoo-tradingbot"]["ssh"]

    bot = paramiko.SSHClient()
    bot.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    bot.connect(bot_ssh["host"], username=bot_ssh["user"], password=bot_ssh.get("password"), timeout=60)
    api_key = fetch_mcp_api_key(bot)
    bot.close()

    agent = paramiko.SSHClient()
    agent.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    agent.connect(agent_ssh["host"], username=agent_ssh["user"], password=agent_ssh.get("password"), timeout=60)

    sftp = agent.open_sftp()
    with sftp.open(REMOTE_CONFIG, "r") as f:
        raw = json.loads(f.read().decode("utf-8"))

    raw["base_url"] = GEEGOO_URL
    raw["geegoo_url"] = GEEGOO_URL
    raw["api_key"] = api_key
    raw["geegoo_api_key"] = api_key
    raw["signal_base_url"] = SIGNAL_URL

    with sftp.open(REMOTE_CONFIG, "w") as f:
        f.write(json.dumps(raw, indent=2, ensure_ascii=False).encode("utf-8") + b"\n")
    sftp.close()

    print(f"Patched {REMOTE_CONFIG}")
    print(f"  geegoo_url = {GEEGOO_URL}")
    print(f"  signal_base_url = {SIGNAL_URL}")
    print(f"  api_key prefix = {api_key[:12]}...")

    doctor_cmd = (
        'export PATH="$HOME/.geegoo/bin:$PATH" GEEGOO_CONFIG="$HOME/.geegoo/config.json"; '
        "geegoo doctor --skip-llm 2>&1"
    )
    code, out = run(agent, doctor_cmd, timeout=120)
    print("\n=== geegoo doctor --skip-llm ===\n", out[-3000:])
    if code != 0:
        agent.close()
        return code

    pre_cmd = (
        'export PATH="$HOME/.geegoo/bin:$PATH" GEEGOO_CONFIG="$HOME/.geegoo/config.json"; '
        "geegoo run pre_market --dry-run 2>&1"
    )
    code, out = run(agent, pre_cmd, timeout=600)
    print("\n=== geegoo run pre_market --dry-run ===\n", out[-4000:])
    agent.close()
    if code != 0:
        return code
    print("\n=== Agent → GeeGoo 3120 verification OK ===")
    return 0


if __name__ == "__main__":
    sys.exit(main())
