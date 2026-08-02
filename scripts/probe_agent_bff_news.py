#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def run(cmd: str) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=120)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def main() -> None:
    print(run("python3 -c \"import json; d=json.load(open('/home/ubuntu/.geegoo/config.json')); print(json.dumps(d.get('data_nodes'), indent=2, ensure_ascii=False))\""))
    print("--- token lens in runtime ---")
    print(run("bash -lc 'pid=$(cat /home/ubuntu/.geegoo/geegoo-agent/agent-runtime.pid); for k in GEEGOO_DATA_CN_TOKEN GEEGOO_DATA_USHK_TOKEN GEEGOO_DATA_SERVICE_TOKEN; do v=$(tr \"\\0\" \"\\n\" < /proc/$pid/environ | grep -m1 \"^$k=\" | cut -d= -f2-); echo $k len=${#v} prefix=${v:0:8}; done'"))
    print("--- BFF news ---")
    for node in ("ashare-cn", "us-hk"):
        print(run(f"curl -s -w '\\nHTTP %{{http_code}}\\n' http://127.0.0.1:3400/v1/data/nodes/{node}/news/sources | tail -3"))
    print("--- overview cache ---")
    print(run("curl -s 'http://127.0.0.1:3400/v1/data/overview?force=true' | python3 -c \"import sys,json;d=json.load(sys.stdin); print([(n['id'], n.get('news',{})) for n in d.get('nodes',[])])\""))


if __name__ == "__main__":
    main()
