#!/usr/bin/env python3
"""Bootstrap CN GeeGooData git repo and start news-worker."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    data_ssh = cfg["targets"]["geegoo-data"]["ssh"]
    cn = cfg["targets"]["geegoo-data-cn"]
    ssh = cn["ssh"]
    rd = cn["remote_dir"]

    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=90)

    dc = paramiko.SSHClient()
    dc.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    dc.connect(data_ssh["host"], username=data_ssh["user"], password=data_ssh["password"], timeout=60)
    _, o, _ = dc.exec_command("grep ^GEEGOO_BOT_SERVICE_API_KEY= /root/apps/GeeGooData/.env | cut -d= -f2-")
    bot_key = o.read().decode().strip()
    dc.close()

    cmd = (
        f"mkdir -p {rd} && cd {rd} && "
        "(test -d .git || git clone https://github.com/ghsemail/GeeGooData.git .) && "
        "git fetch origin main && git reset --hard origin/main && "
        "grep -q GEEGOO_DATA_MONGO_URI .env 2>/dev/null || echo GEEGOO_DATA_MONGO_URI=mongodb://127.0.0.1:27017 >> .env && "
        "grep -q GEEGOO_BOT_SERVICE_API_URL .env 2>/dev/null || echo GEEGOO_BOT_SERVICE_API_URL=http://118.195.135.97:3140 >> .env && "
        f"grep -q '^GEEGOO_BOT_SERVICE_API_KEY=' .env 2>/dev/null || echo GEEGOO_BOT_SERVICE_API_KEY={bot_key} >> .env && "
        "grep -q GEEGOO_DATA_NEWS_REFRESH_ENABLED .env 2>/dev/null || echo GEEGOO_DATA_NEWS_REFRESH_ENABLED=true >> .env && "
        "grep -q market_capabilities.cn .env 2>/dev/null || echo GEEGOO_DATA_MARKET_CAPABILITIES_FILE=config/market_capabilities.cn.xml >> .env && "
        "grep -q news_sources.cn .env 2>/dev/null || echo GEEGOO_DATA_NEWS_SOURCES_FILE=config/news_sources.cn.xml >> .env && "
        "bash start.sh restart && sleep 3 && bash start.sh status"
    )
    print("$", cmd[:120], "...")
    _, o, e = c.exec_command(cmd, timeout=600)
    print((o.read() + e.read()).decode())
    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
