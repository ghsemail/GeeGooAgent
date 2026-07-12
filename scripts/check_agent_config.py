#!/usr/bin/env python3
import json
from pathlib import Path
import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main():
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, _ = c.exec_command(
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print('geegoo_url',c.get('geegoo_url')); print('base_url',c.get('base_url')); "
        "print('signal',c.get('signal_base_url')); print('data',c.get('data_base_url'))\"",
        timeout=30,
    )
    print(o.read().decode())


if __name__ == "__main__":
    main()
