#!/usr/bin/env python3
"""Probe Monday routes on GeeGooSignal catalog-api after deploy."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
MONDAY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\monday\config.json")

ENDPOINTS = [
    "/getSinglePromptTemplate",
    "/getSinglePromptTemplateByIndex",
    "/addSinglePromptTemplate",
    "/editPromptTemplate",
    "/deletePromptTemplate",
    "/switchPromptStatus",
    "/getCustomSignal",
    "/getCustomSignalForSkill",
    "/getAllCustomSignalId",
    "/getCustomStrategyDefinitions",
    "/addCustomSignal",
    "/editCustomSignal",
    "/deleteCustomSignal",
]


def ssh_run(client: paramiko.SSHClient, cmd: str) -> str:
    _, stdout, stderr = client.exec_command(cmd, timeout=60)
    out = stdout.read().decode("utf-8", errors="replace").strip()
    err = stderr.read().decode("utf-8", errors="replace").strip()
    return out or err


def read_key(client: paramiko.SSHClient, path: str, key: str) -> str:
    return ssh_run(client, f"grep '^{key}=' {path} 2>/dev/null | head -1 | cut -d= -f2-").strip().strip('"')


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    sig = cfg["targets"]["geegoo-signal"]["ssh"]
    agent = cfg["targets"]["geegoo-agent"]["ssh"]

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(sig["host"], sig.get("port", 22), sig["user"], sig.get("password"), timeout=30)

    catalog_key = read_key(client, "/root/apps/GeeGooSignal/.env", "GEEGOO_SIGNAL_CATALOG_API_KEY")
    qt_db = read_key(client, "/root/apps/GeeGooSignal/.env", "GEEGOO_QT_MONGO_DB") or "(unset)"

    mcp_token = ""
    if MONDAY_CFG.exists():
        mcp_token = json.loads(MONDAY_CFG.read_text(encoding="utf-8")).get("mcp_token", "")

    agent_client = paramiko.SSHClient()
    agent_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    agent_client.connect(agent["host"], agent.get("port", 22), agent["user"], agent.get("password"), timeout=30)
    agent_token = ssh_run(
        agent_client,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); print(c.get('mcp_token',''))\"",
    ).strip()
    agent_client.close()
    if agent_token:
        mcp_token = agent_token

    body = json.dumps({"mcp_token": mcp_token, "type": "tech", "index": "EMA", "period": "daily"})
    auth = f"-H 'Authorization: Bearer {catalog_key}'" if catalog_key else ""

    print(f"catalog_key={'set' if catalog_key else 'MISSING'} qt_db={qt_db} mcp_token={'set' if mcp_token else 'MISSING'}")
    print()

    unknown = ssh_run(client, f"curl -sS -m 8 -o /dev/null -w '%{{http_code}}' {auth} -H 'Content-Type: application/json' -d '{body}' http://127.0.0.1:3210/__monday_unknown__")
    print(f"  {'(unknown)':40} {unknown}")

    for ep in ENDPOINTS:
        code = ssh_run(
            client,
            f"curl -sS -m 12 -o /dev/null -w '%{{http_code}}' {auth} -H 'Content-Type: application/json' -d '{body}' http://127.0.0.1:3210{ep}",
        )
        print(f"  {ep:40} {code}")

    # sample response for definitions
    sample = ssh_run(
        client,
        f"curl -sS -m 12 {auth} -H 'Content-Type: application/json' -d '{json.dumps({'mcp_token': mcp_token})}' http://127.0.0.1:3210/getCustomStrategyDefinitions | head -c 200",
    )
    print("\ngetCustomStrategyDefinitions sample:", sample)

    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
