#!/usr/bin/env python3
"""Probe CN quotes from internal servers."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target: str, cmd: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=60)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> None:
    code = "000858.SZ"
    for target, from_host in [
        ("geegoo-bot", "bot"),
        ("geegoo-tradingsignal", "signal"),
        ("geegoo-data-cn", "cn-local"),
    ]:
        print(f"\n## from {from_host}")
        print(
            run(
                target,
                f"curl -s -m 20 -X POST http://82.157.97.76:3300/v1/market/quote "
                f"-H 'Content-Type: application/json' -d '{{\"code\":\"{code}\"}}'",
            )
        )
        print(
            run(
                target,
                f"curl -s -m 20 -X POST http://82.157.97.76:3300/v1/market/klines "
                f"-H 'Content-Type: application/json' -d '{{\"code\":\"{code}\",\"frequency\":\"daily\",\"limit\":5}}'",
            )[:400]
        )
        print(
            run(
                target,
                f"curl -s -m 20 -X POST http://47.80.14.120:3300/v1/market/quote "
                f"-H 'Content-Type: application/json' -d '{{\"code\":\"{code}\"}}'",
            )
        )

    print("\n## CN node env")
    print(run("geegoo-data-cn", "grep -E 'MARKET_CAP|FUTU|OPEND|PROVIDER|TOKEN' /home/ubuntu/apps/GeeGooData/.env | head -20"))
    print(run("geegoo-data-cn", "pgrep -af 'data-api|usdata|futu|helper' || true"))
    print(run("geegoo-data-cn", "tail -20 /home/ubuntu/apps/GeeGooData/data-api.out"))


if __name__ == "__main__":
    main()
