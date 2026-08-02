#!/usr/bin/env python3
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
CLIENT_KEY = (
    "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
)


def probe(url: str, key: str) -> None:
    req = urllib.request.Request(
        url,
        headers={
            "Authorization": f"Bearer {key}",
            "X-Client-Source": "probe",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            body = resp.read(180)
            print(f"OK {url} -> {resp.status} {body[:120]!r}")
    except urllib.error.HTTPError as e:
        print(f"HTTP {e.code} {url} -> {e.read()[:160]!r}")
    except Exception as e:
        print(f"ERR {url} -> {e}")


def main() -> None:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    bot = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(bot["host"], username=bot["user"], password=bot.get("password"), timeout=60)

    _, o, _ = c.exec_command(
        "grep AGENT_API /home/ubuntu/apps/GeeGooBot/.env | head -5", timeout=30
    )
    print("=== GeeGooBot .env ===")
    print(o.read().decode())

    _, o, _ = c.exec_command(
        "KEY=$(grep '^GEEGOO_BOT_AGENT_API_KEY=' /home/ubuntu/apps/GeeGooBot/.env | cut -d= -f2-); "
        "curl -s -o /tmp/dash.out -w '%{http_code}' -H \"Authorization: Bearer $KEY\" "
        "http://127.0.0.1:3110/op_agent/v1/dashboard/data; echo; head -c 120 /tmp/dash.out; echo",
        timeout=30,
    )
    print("=== local BFF dashboard with server key ===")
    print(o.read().decode())

    c.close()

    print("\n=== external probes ===")
    probe("http://118.195.135.97:3110/op_agent/v1/dashboard/data", CLIENT_KEY)
    probe("http://146.56.225.252:8088/op_agent/v1/dashboard/data", CLIENT_KEY)

    sig = cfg["targets"]["geegoo-signal"]["ssh"]
    c2 = paramiko.SSHClient()
    c2.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c2.connect(sig["host"], username=sig["user"], password=sig.get("password"), timeout=60)
    _, o, _ = c2.exec_command(
        "docker ps --format '{{.Names}}' | head -3; "
        "docker exec $(docker ps -q --filter publish=8088 | head -1) "
        "nginx -T 2>/dev/null | grep -n op_agent | head -15",
        timeout=40,
    )
    print("\n=== nginx op_agent on :8088 ===")
    print(o.read().decode() or "(no op_agent routes)")
    c2.close()


if __name__ == "__main__":
    main()
