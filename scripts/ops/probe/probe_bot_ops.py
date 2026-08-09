#!/usr/bin/env python3
"""Probe GeeGooBot app-api ops routes."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    for target in ("geegoo-bot", "geegoo-tradingsignal"):
        ssh_cfg = cfg["targets"][target]["ssh"]
        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        client.connect(
            hostname=ssh_cfg["host"],
            port=int(ssh_cfg.get("port", 22)),
            username=ssh_cfg["user"],
            password=ssh_cfg.get("password"),
            timeout=60,
        )

        def run(cmd: str) -> str:
            _, o, e = client.exec_command(cmd)
            return (o.read() + e.read()).decode().strip()

        print(f"\n=== {target} ({ssh_cfg['host']}) ===")
        if target == "geegoo-bot":
            for path in ("/queryTradingDate", "/getUserList", "/getKeyList"):
                out = run(
                    f"curl -s -w '\\nHTTP %{{http_code}}' "
                    f"-H 'Authorization: Bearer {BOT_KEY}' "
                    f"-H 'Content-Type: application/json' -d '{{}}' "
                    f"http://127.0.0.1:3100{path}"
                )
                print(path, "->", out[:400])
        else:
            out = run(
                f"curl -s -w '\\nHTTP %{{http_code}}' "
                f"-H 'Authorization: Bearer {BOT_KEY}' "
                f"-H 'Content-Type: application/json' -d '{{}}' "
                f"http://127.0.0.1:8088/op_bot/queryTradingDate"
            )
            print("/op_bot/queryTradingDate ->", out[:400])
        client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
