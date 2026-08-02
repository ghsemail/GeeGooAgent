#!/usr/bin/env python3
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
TOKEN = "mcp_HVTSYfumrCexAU66EutTM4v2A5aGYXiF"
API_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def probe(url: str) -> None:
    req = urllib.request.Request(
        url,
        headers={
            "Authorization": f"Bearer {API_KEY}",
            "X-MCP-Token": TOKEN,
            "X-Client-Source": "verify",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            print("OK", url, resp.status, resp.read(100))
    except urllib.error.HTTPError as e:
        print("HTTP", url, e.code, e.read()[:160])


def main() -> None:
    probe("http://118.195.135.97:3110/op_agent/v1/dashboard/data")
    probe("http://146.56.225.252:8088/op_agent/v1/dashboard/data")

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-signal"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=60)
    cmd = (
        "curl -s -o /tmp/d.out -w '%{http_code}' "
        "-H 'Authorization: Bearer " + API_KEY + "' "
        "-H 'X-MCP-Token: " + TOKEN + "' "
        "http://127.0.0.1:8088/op_agent/v1/dashboard/data; echo; head -c 120 /tmp/d.out; echo"
    )
    _, o, _ = c.exec_command(cmd, timeout=30)
    print("local nginx:", o.read().decode())
    _, o, _ = c.exec_command(
        "docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}' | "
        "xargs -I{} docker exec {} nginx -T 2>/dev/null | grep -A20 'location /op_agent/'",
        timeout=40,
    )
    print("nginx block:\n", o.read().decode())
    c.close()


if __name__ == "__main__":
    main()
