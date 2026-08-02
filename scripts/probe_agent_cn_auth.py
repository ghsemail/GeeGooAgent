#!/usr/bin/env python3
"""Debug CN news auth from agent server."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
AGENT_DIR = "/home/ubuntu/.geegoo/geegoo-agent"


def run(target: str, cmd: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=120)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> None:
    print("=== find config ===")
    print(run("geegoo-agent", "find /home/ubuntu/.geegoo -name 'config.json' 2>/dev/null; ls -la /home/ubuntu/.geegoo/"))
    print("=== start.sh env loading ===")
    print(run("geegoo-agent", f"grep -n 'env\\|ENV\\|dotenv' {AGENT_DIR}/start.sh | head -30"))
    print("=== runtime pid env ===")
    print(run("geegoo-agent", "pid=$(cat /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.pid 2>/dev/null); tr '\\0' '\\n' < /proc/$pid/environ 2>/dev/null | grep GEEGOO_DATA | sed 's/=.*/=***/'"))
    print("=== direct curl CN from agent ===")
    print(run("geegoo-agent", "source /home/ubuntu/.geegoo/geegoo-agent/.env 2>/dev/null; echo CN_LEN=${#GEEGOO_DATA_CN_TOKEN}; curl -s -w '\\nHTTP %{http_code}\\n' -H \"Authorization: Bearer $GEEGOO_DATA_CN_TOKEN\" http://82.157.97.76:3300/v1/news/sources | tail -5"))
    print("=== CN token on data-cn host ===")
    print(run("geegoo-data-cn", "grep ^GEEGOO_DATA_SERVICE_TOKEN= /home/ubuntu/apps/GeeGooData/.env | wc -c"))
    print("=== compare first 8 chars ===")
    print(run("geegoo-agent", "source /home/ubuntu/.geegoo/geegoo-agent/.env; echo AGENT=${GEEGOO_DATA_CN_TOKEN:0:8}"))
    print(run("geegoo-data-cn", "t=$(grep ^GEEGOO_DATA_SERVICE_TOKEN= /home/ubuntu/apps/GeeGooData/.env|cut -d= -f2-); echo DATA=${t:0:8}"))


if __name__ == "__main__":
    main()
