#!/usr/bin/env python3
"""Diagnose Agent 401: trace which layer rejects the request."""
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
TOKEN = "mcp_HVTSYfumrCexAU66EutTM4v2A5aGYXiF"
KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def probe(url: str, headers: dict) -> None:
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            print(f"OK {url} {dict(headers).keys()} -> {r.status}")
    except urllib.error.HTTPError as e:
        print(f"HTTP {e.code} {url} keys={list(headers)} -> {e.read()[:120]!r}")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    cmd = (
        "KEY=$(grep '^GEEGOO_BOT_AGENT_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env 2>/dev/null | cut -d= -f2-); "
        "TOK='mcp_HVTSYfumrCexAU66EutTM4v2A5aGYXiF'; "
        "echo '--- local 8088 mcp only ---'; "
        "curl -s -o /dev/null -w '%{http_code}' -H \"X-MCP-Token: $TOK\" http://127.0.0.1:8088/op_agent/v1/dashboard/data; echo; "
        "echo '--- local 8088 both ---'; "
        "curl -s -o /dev/null -w '%{http_code}' -H \"Authorization: Bearer $KEY\" -H \"X-MCP-Token: $TOK\" http://127.0.0.1:8088/op_agent/v1/dashboard/data; echo; "
        "echo '--- bot 3110 both ---'; "
        "curl -s -o /dev/null -w '%{http_code}' -H \"Authorization: Bearer $KEY\" -H \"X-MCP-Token: $TOK\" http://118.195.135.97:3110/op_agent/v1/dashboard/data; echo; "
        "tail -3 /home/ubuntu/apps/GeeGooBot/agent-api.out 2>/dev/null || tail -3 /root/apps/GeeGooBot/agent-api.out 2>/dev/null || echo no log"
    )
    _, o, e = c.exec_command(cmd, timeout=40)
    print(o.read().decode())
    if e.read().decode().strip():
        print("stderr:", e.read().decode())
    c.close()

    print("\n=== external ===")
    probe(
        "http://146.56.225.252:8088/op_agent/v1/dashboard/data",
        {"X-MCP-Token": TOKEN},
    )
    probe(
        "http://146.56.225.252:8088/op_agent/v1/dashboard/data",
        {"Authorization": f"Bearer {KEY}", "X-MCP-Token": TOKEN, "X-Client-Source": "trading_operation"},
    )
    probe(
        "http://146.56.225.252:8088/op_agent/v1/dashboard/data",
        {"Authorization": f"Bearer {KEY}", "X-MCP-Token": "mcp_invalid_token"},
    )


if __name__ == "__main__":
    main()
