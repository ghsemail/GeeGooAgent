#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(target, cmd):
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=30)
    print(o.read().decode() or e.read().decode())


def main():
    run("geegoo-tradingbot", "ss -tlnp | grep 3140 || netstat -tlnp 2>/dev/null | grep 3140")
    run("geegoo-tradingbot", "ps aux | grep service-api | grep -v grep")
    run("geegoo-tradingbot", "grep MONGO /home/ubuntu/apps/GeeGooBot/.env")
    run("geegoo-tradingbot", "tail -5 /home/ubuntu/apps/GeeGooBot/service-api.out")


if __name__ == "__main__":
    main()
