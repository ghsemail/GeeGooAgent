#!/usr/bin/env python3
"""Probe CN vs US-HK news API auth and agent runtime data node tokens."""
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def http_get(url: str, bearer: str = "") -> tuple[int, str]:
    headers = {}
    if bearer:
        headers["Authorization"] = f"Bearer {bearer}"
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, r.read(200).decode("utf-8", "replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read(200).decode("utf-8", "replace")


def main() -> None:
    print("=== direct GeeGooData ===")
    for label, url in [
        ("CN", "http://82.157.97.76:3300/v1/news/sources"),
        ("US-HK", "http://47.80.14.120:3300/v1/news/sources"),
    ]:
        code, body = http_get(url)
        print(f"{label} no-auth: {code} {body[:80]}")

    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
    _, o, _ = c.exec_command(
        "grep -E 'GEEGOO_DATA_(CN|USHK|SERVICE)_TOKEN' "
        "/home/ubuntu/.geegoo/geegoo-agent/.env 2>/dev/null || true",
        timeout=30,
    )
    print("\n=== agent .env token keys ===")
    print(o.read().decode())
    _, o, _ = c.exec_command("cat /home/ubuntu/.geegoo/geegoo-agent/config.json", timeout=30)
    raw = o.read().decode()
    try:
        conf = json.loads(raw)
        print("\n=== data_nodes ===")
        print(json.dumps(conf.get("data_nodes", []), indent=2, ensure_ascii=False)[:800])
    except json.JSONDecodeError:
        print(raw[:400])
    c.close()


if __name__ == "__main__":
    main()
