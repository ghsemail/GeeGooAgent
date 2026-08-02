#!/usr/bin/env python3
"""Verify getUserStockTrend after signal key fix."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    bot = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(bot["host"], username=bot["user"], password=bot.get("password"), timeout=60)

    def run(cmd: str) -> str:
        _, o, e = c.exec_command(cmd)
        return (o.read() + e.read()).decode().strip()

    app_key = run("grep '^GEEGOO_BOT_APP_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env | cut -d= -f2-")
    # user with stocks from earlier probe
    body = (
        '{"user_id":"67935cda6272feb48b49ba49","type":"flag","frequency":"5m",'
        '"signal_index_list":["macd"],"language":"cn"}'
    )
    out = run(
        f"curl -s -w '\\nHTTP %{{http_code}}' "
        f"-H 'Authorization: Bearer {app_key}' "
        f"-H 'Content-Type: application/json' "
        f"-d '{body}' "
        f"http://127.0.0.1:3100/getUserStockTrend"
    )
    print(out[:800])
    c.close()


if __name__ == "__main__":
    main()
