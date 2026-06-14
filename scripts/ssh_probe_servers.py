#!/usr/bin/env python3
"""Probe Signal and Bot servers via SSH (read-only diagnostics)."""

from __future__ import annotations

import os
import sys

import paramiko

USER = os.environ.get("SSH_USER", "ubuntu")
PASS = os.environ.get("SSH_PASS", "")

SERVERS = {
    "signal": ("43.134.94.87", "Signal"),
    "bot": ("118.195.135.97", "Bot/MCP"),
}


def run(client: paramiko.SSHClient, cmd: str) -> str:
    print(f"--- {cmd[:100]}")
    _stdin, stdout, stderr = client.exec_command(cmd, timeout=45)
    out = stdout.read().decode("utf-8", errors="replace").strip()
    err = stderr.read().decode("utf-8", errors="replace").strip()
    if out:
        print(out[:4000])
    if err and "grep" not in err and "curl" not in err:
        print("ERR:", err[:500])
    return out


def main() -> int:
    if not PASS:
        print("Set SSH_PASS env var", file=sys.stderr)
        return 1

    signal_cmds = [
        "ss -tlnp | grep -E '5600|5700|5800|7000|5100|5200'",
        "ls -la /home/ubuntu/apps/TradingServer/ 2>/dev/null | head -15",
        "head -35 /home/ubuntu/apps/TradingServer/app/main.py 2>/dev/null || ls /home/ubuntu/apps/TradingServer/",
        "curl -s http://127.0.0.1:7000/docs 2>/dev/null | head -3 || curl -s -o /dev/null -w '7000 HTTP %{http_code}' http://127.0.0.1:7000/",
    ]

    bot_cmds = [
        "cat /home/ubuntu/apps/TradingBot/Config/APIConnection.py | head -40",
        "ss -tlnp | grep python",
        "grep -n 'app.run\\|port\\|5600\\|5700\\|5900\\|6200\\|6300' /home/ubuntu/apps/TradingBot/mcpAPIServer.py /home/ubuntu/apps/TradingBot/botAPIServer.py /home/ubuntu/apps/TradingBot/marketAPIServer.py /home/ubuntu/apps/TradingBot/UtilityServer.py 2>/dev/null | head -30",
        "curl -s http://127.0.0.1:7000/searchCode -X POST -H 'Content-Type: application/json' -d '{\"regex\":\"00700\"}' 2>/dev/null | head -c 200 || echo 'no 7000'",
    ]

    for key, (host, label) in SERVERS.items():
        print("=" * 60)
        print(f"{label} @ {host}")
        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        try:
            client.connect(host, username=USER, password=PASS, timeout=25)
            run(client, "hostname")
            for cmd in signal_cmds if key == "signal" else bot_cmds:
                run(client, cmd)
            client.close()
        except Exception as exc:
            print("CONNECT FAIL:", exc)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
