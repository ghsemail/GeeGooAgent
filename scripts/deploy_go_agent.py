#!/usr/bin/env python3
"""Deploy Go GeeGooAgent to geegoo-agent server; wire config to GeeGoo 31xx/32xx/33xx."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
GEEGOO_BOT_MCP = "http://118.195.135.97:3120"
GEEGOO_SIGNAL = "http://146.56.225.252:3210"
GEEGOO_DATA = "http://47.80.14.120:3300"
ALLOWED_HOSTS = [
    "118.195.135.97",
    "146.56.225.252",
    "47.80.14.120",
    "127.0.0.1",
    "localhost",
]


def load_deploy() -> dict:
    return json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))


def ssh_connect(target: dict) -> paramiko.SSHClient:
    s = target["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    return c


def run(client: paramiko.SSHClient, cmd: str, timeout: int = 600) -> tuple[int, str]:
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = (stdout.read() + stderr.read()).decode("utf-8", errors="replace")
    return stdout.channel.recv_exit_status(), out


def fetch_mcp_api_key() -> str:
    cfg = load_deploy()
    bot = ssh_connect(cfg["targets"]["geegoo-tradingbot"])
    _, o, _ = bot.exec_command(
        "grep '^GEEGOO_BOT_MCP_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env 2>/dev/null | cut -d= -f2-",
        timeout=30,
    )
    key = o.read().decode().strip()
    if not key:
        _, o, _ = bot.exec_command(
            "python3 -c \"import re; t=open('/home/ubuntu/apps/TradingBot/mcp/constants.py').read(); "
            "m=re.search(r'API_KEY\\s*=\\s*\\\"([^\\\"]+)\\\"', t); print(m.group(1) if m else '')\"",
            timeout=30,
        )
        key = o.read().decode().strip()
    bot.close()
    if not key:
        local = Path(r"D:\Geegoo\TradingBot\mcp\constants.py")
        if local.is_file():
            m = re.search(r'API_KEY\s*=\s*["\']([^"\']+)["\']', local.read_text(encoding="utf-8"))
            key = m.group(1) if m else ""
    if not key:
        raise RuntimeError("cannot resolve GeeGooBot MCP API_KEY")
    return key


def patch_config(raw: dict, api_key: str) -> dict:
    raw["base_url"] = GEEGOO_BOT_MCP
    raw["geegoo_url"] = GEEGOO_BOT_MCP
    raw["api_key"] = api_key
    raw["geegoo_api_key"] = api_key
    raw["signal_base_url"] = GEEGOO_SIGNAL
    raw["data_base_url"] = GEEGOO_DATA
    sandbox = raw.setdefault("sandbox", {})
    hosts = set(sandbox.get("allowed_hosts") or [])
    hosts.update(ALLOWED_HOSTS)
    sandbox["allowed_hosts"] = sorted(hosts)
    return raw


def write_agent_env(sftp: paramiko.SFTPClient, config_path: str) -> None:
    env_lines = f"""export GEEGOO_BOT_MCP_URL={GEEGOO_BOT_MCP}
export GEEGOO_SIGNAL_CATALOG_API_URL={GEEGOO_SIGNAL}
export GEEGOO_DATA_HTTP_URL={GEEGOO_DATA}
export GEEGOO_CONFIG={config_path}
export PATH=/home/ubuntu/.geegoo/bin:/usr/local/go/bin:$PATH
"""
    with sftp.open("/home/ubuntu/.geegoo/agent.env", "w") as f:
        f.write(env_lines)


def main() -> int:
    cfg = load_deploy()
    agent_target = cfg["targets"]["geegoo-agent"]
    api_key = fetch_mcp_api_key()
    print(f"MCP API key prefix: {api_key[:12]}...")

    agent = ssh_connect(agent_target)
    install_dir = agent_target["remote_dir"]
    config_path = "/home/ubuntu/.geegoo/config.json"

    steps = [
        f"test -d {install_dir}/.git || git clone git@github.com:ghsemail/GeeGooAgent.git {install_dir}",
        f"cd {install_dir} && git fetch origin main && git reset --hard origin/main",
        f"cd {install_dir} && bash scripts/ensure-go.sh",
        f"cd {install_dir} && bash scripts/install-go.sh",
    ]
    for cmd in steps:
        print(f"\n>>> {cmd}")
        code, out = run(agent, cmd, timeout=900)
        print(out[-2500:] if len(out) > 2500 else out)
        if code != 0:
            agent.close()
            return code

    sftp = agent.open_sftp()
    try:
        with sftp.open(config_path, "r") as f:
            raw = json.loads(f.read().decode("utf-8"))
    except FileNotFoundError:
        raw = json.loads(Path(r"D:\Geegoo\GeeGooAgent\config.example.json").read_text(encoding="utf-8"))
        raw["output_dir"] = "/home/ubuntu/.geegoo/data"

    raw = patch_config(raw, api_key)
    write_agent_env(sftp, config_path)
    with sftp.open(config_path, "w") as f:
        f.write(json.dumps(raw, indent=2, ensure_ascii=False).encode("utf-8") + b"\n")
    sftp.close()
    print(f"\nPatched {config_path}")
    print(f"  geegoo_url = {GEEGOO_BOT_MCP}")
    print(f"  signal_base_url = {GEEGOO_SIGNAL}")
    print(f"  data_base_url = {GEEGOO_DATA}")

    code, out = run(
        agent,
        f'cd {install_dir} && bash start.sh restart-runtime',
        timeout=120,
    )
    print("\n=== start-runtime ===\n", out)

    doctor_cmd = (
        'export PATH="$HOME/.geegoo/bin:/usr/local/go/bin:$PATH" '
        f'GEEGOO_CONFIG="{config_path}"; geegoo doctor 2>&1'
    )
    code, out = run(agent, doctor_cmd, timeout=180)
    print("\n=== geegoo doctor ===\n", out)

    # verify no legacy ports in config
    _, out = run(agent, f'grep -E "5700|5800|5600" {config_path} || echo NO_LEGACY_PORTS')
    print("\n=== legacy port scan ===\n", out.strip())

    agent.close()
    if code != 0:
        return code
    print("\n=== GeeGooAgent Go deployed; backends = GeeGoo 31xx/32xx/33xx ===")
    return 0


if __name__ == "__main__":
    sys.exit(main())
