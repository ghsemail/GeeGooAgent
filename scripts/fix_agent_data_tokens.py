#!/usr/bin/env python3
"""Sync GeeGooData service tokens into ~/.geegoo/agent.env and restart agent-runtime."""
from __future__ import annotations

import base64
import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
AGENT_ENV = "/home/ubuntu/.geegoo/agent.env"
AGENT_DIR = "/home/ubuntu/.geegoo/geegoo-agent"


def run(target: str, cmd: str, timeout: int = 120) -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"][target]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)
    _, o, e = c.exec_command(cmd, timeout=timeout)
    out = (o.read() + e.read()).decode("utf-8", errors="replace")
    c.close()
    return out


def upsert_agent_env(c: paramiko.SSHClient, key: str, value: str) -> None:
    b64 = base64.b64encode(value.encode()).decode()
    py = f"""
import base64, pathlib, re
p = pathlib.Path({json.dumps(AGENT_ENV)})
key = {json.dumps(key)}
val = base64.b64decode({json.dumps(b64)}).decode()
lines = p.read_text(encoding='utf-8').splitlines() if p.exists() else []
pat = re.compile(r'^\\s*(?:export\\s+)?' + re.escape(key) + r'=')
out, found = [], False
for line in lines:
    if pat.match(line):
        out.append(f'export {{key}}={{val}}')
        found = True
    else:
        out.append(line)
if not found:
    out.append(f'export {{key}}={{val}}')
p.parent.mkdir(parents=True, exist_ok=True)
p.write_text('\\n'.join(out) + ('\\n' if out else ''), encoding='utf-8')
print('ok', key, val[:8])
"""
    _, o, e = c.exec_command(f"python3 <<'PY'\n{py}\nPY", timeout=60)
    print((o.read() + e.read()).decode())


def main() -> int:
    cn_token = run("geegoo-data-cn", "grep ^GEEGOO_DATA_SERVICE_TOKEN= /home/ubuntu/apps/GeeGooData/.env | cut -d= -f2-").strip()
    ushk_token = run("geegoo-data", "grep ^GEEGOO_DATA_SERVICE_TOKEN= /root/apps/GeeGooData/.env | cut -d= -f2-").strip()
    if not cn_token or not ushk_token:
        raise SystemExit("missing data node tokens")

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=60)

    sed_mask = r"sed 's/\(=.\{8\}\).*/\1***/'"
    print("before:")
    print(run("geegoo-agent", f"grep GEEGOO_DATA_ /home/ubuntu/.geegoo/agent.env | {sed_mask}"))

    upsert_agent_env(c, "GEEGOO_DATA_CN_TOKEN", cn_token)
    upsert_agent_env(c, "GEEGOO_DATA_USHK_TOKEN", ushk_token)

    print("after:")
    print(run("geegoo-agent", f"grep GEEGOO_DATA_ /home/ubuntu/.geegoo/agent.env | {sed_mask}"))

    # restart with agent.env exported so runtime inherits correct tokens
    print(run("geegoo-agent", f"bash -lc 'set -a; source {AGENT_ENV}; set +a; cd {AGENT_DIR} && bash start.sh restart-runtime && sleep 2'"))
    print(run("geegoo-agent", "curl -sf http://127.0.0.1:3400/health && echo"))

    for node in ("ashare-cn", "us-hk"):
        out = run("geegoo-agent", f"curl -s -w '\\nHTTP %{{http_code}}\\n' http://127.0.0.1:3400/v1/data/nodes/{node}/news/sources | tail -2")
        print(f"{node}: {out.strip()}")

    c.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
