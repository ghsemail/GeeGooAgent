#!/usr/bin/env python3
"""Compare signal-api keys on GeeGooSignal vs GeeGooBot outbound."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(ssh_cfg: dict, cmd: str) -> str:
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=60,
    )
    _, o, e = c.exec_command(cmd)
    out = (o.read() + e.read()).decode()
    c.close()
    return out.strip()


def grep_env(ssh_cfg: dict, path: str, key: str) -> str:
    return run(ssh_cfg, f"grep '^{key}=' {path} 2>/dev/null | cut -d= -f2- || true")


def probe_signal(host_ssh: dict, key: str) -> str:
    return run(
        host_ssh,
        f"curl -s -o /dev/null -w '%{{http_code}}' "
        f"-H 'Authorization: Bearer {key}' "
        f"-H 'Content-Type: application/json' "
        f"-d '{{\"code_list\":[\"00700.HK\"],\"type\":\"flag\",\"frequency\":\"5m\",\"signal_index_list\":[],\"language\":\"cn\"}}' "
        f"http://127.0.0.1:3200/getCodeListFlag",
    )


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    sig = cfg["targets"]["geegoo-tradingsignal"]["ssh"]
    bot = cfg["targets"]["geegoo-bot"]["ssh"]

    sig_key = grep_env(sig, "/root/apps/GeeGooSignal/.env", "GEEGOO_SIGNAL_SIGNAL_API_KEY")
    bot_key = grep_env(bot, "/home/ubuntu/apps/GeeGooBot/.env", "GEEGOO_SIGNAL_SIGNAL_API_KEY")
    mcp_key = grep_env(bot, "/home/ubuntu/apps/GeeGooBot/.env", "GEEGOO_BOT_MCP_API_KEY")

    print("GeeGooSignal SIGNAL_API_KEY:", sig_key[:20] + "..." if sig_key else "(empty)")
    print("GeeGooBot  SIGNAL_API_KEY:", bot_key[:20] + "..." if bot_key else "(empty)")
    print("GeeGooBot  MCP_API_KEY:    ", mcp_key[:20] + "..." if mcp_key else "(empty)")
    print("keys match:", sig_key == bot_key)

    app_key = "a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402"
    sk_key = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
    for label, key in [("app hardcoded", app_key), ("bot sk key", sk_key), ("bot env signal", bot_key)]:
        if key:
            print(f"probe {label}: HTTP {probe_signal(sig, key)}")


if __name__ == "__main__":
    main()
