#!/usr/bin/env python3
import json, time
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
UID = "64afddf8c2a269ac1846fe70"
BODY = json.dumps({"user_id": UID})


def ssh(cmd, timeout=120):
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-tradingbot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = o.read().decode("utf-8", errors="replace")
    err = e.read().decode("utf-8", errors="replace")
    code = o.channel.recv_exit_status()
    c.close()
    print(out or err)
    if code != 0:
        raise SystemExit(code)


def main():
    ssh("cd /home/ubuntu/apps/GeeGooBot && bash start.sh build")
    ssh("cd /home/ubuntu/apps/GeeGooBot && bash start.sh stop && sleep 1 && echo 2 | bash start.sh")
    time.sleep(3)
    ssh(f"curl -s -X POST http://127.0.0.1:3140/checkUser -H 'Content-Type: application/json' -d '{BODY}'")
    ssh("tail -3 /home/ubuntu/apps/GeeGooBot/service-api.out")


if __name__ == "__main__":
    main()
