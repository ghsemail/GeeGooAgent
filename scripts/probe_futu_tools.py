#!/usr/bin/env python3
"""Probe getTicker / getBroker / getPosition on GeeGooBot mcp-api."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 90) -> tuple[int, str, str]:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    return code, out, err


def http_code(text: str) -> str:
    m = re.search(r"HTTP:(\d{3})", text)
    return m.group(1) if m else "?"


def summarize(path: str, raw: str) -> str:
    http = http_code(raw)
    body = re.sub(r"\nHTTP:\d{3}$", "", raw).strip()
    note = f"HTTP {http}"
    try:
        data = json.loads(body)
    except json.JSONDecodeError:
        return f"{note} | non-JSON: {body[:200]}"
    api_code = data.get("code")
    msg = data.get("message", "")
    payload = data.get("data")
    if isinstance(payload, list):
        return f"{note} | api_code={api_code} | items={len(payload)} | {msg}"
    if isinstance(payload, dict):
        keys = list(payload.keys())[:8]
        return f"{note} | api_code={api_code} | data_keys={keys} | {msg}"
    return f"{note} | api_code={api_code} | data={payload!r} | {msg}"


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent_ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    bot_ssh = cfg["targets"]["geegoo-bot"]["ssh"]

    _, out, _ = ssh_run(
        agent_ssh,
        "python3 -c \"import json; c=json.load(open('/home/ubuntu/.geegoo/config.json')); "
        "print(c.get('mcp_token','')); print(c.get('geegoo_api_key',''))\"",
    )
    lines = [ln.strip() for ln in out.strip().splitlines() if ln.strip()]
    token = lines[0] if lines else ""
    api_key = lines[1] if len(lines) > 1 else ""
    if not token or not api_key:
        print("ERROR: missing mcp_token or geegoo_api_key on agent host")
        return 1

    print("Futu MCP probe (GeeGooBot localhost:3120)")
    print(f"  test user token: {token[:8]}...")

    cases = [
        ("getTicker", {"mcp_token": token, "code": "00700.HK", "num": 5}),
        ("getBroker", {"mcp_token": token, "code": "00700.HK"}),
        ("getPosition", {"mcp_token": token, "code": "00700.HK"}),
    ]
    ok = 0
    for path, body in cases:
        payload = json.dumps(body, ensure_ascii=False)
        cmd = (
            f"curl -sS -m 45 -w '\\nHTTP:%{{http_code}}' "
            f"-H 'Authorization: Bearer {api_key}' -H 'Content-Type: application/json' "
            f"-d '{payload}' http://127.0.0.1:3120/{path}"
        )
        _, raw, err = ssh_run(bot_ssh, cmd, timeout=60)
        if not raw.strip() and err.strip():
            raw = err
        summary = summarize(path, raw)
        print(f"  {path:14} {summary}")
        if "api_code=100" in summary and ("items=" in summary and "items=0" not in summary or "data_keys=" in summary):
            ok += 1
        elif "api_code=100" in summary and "items=0" in summary:
            print("    -> code=100 but empty list (may be non-trading hours / not subscribed)")

    print(f"\nResult: {ok}/3 returned non-empty data")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
