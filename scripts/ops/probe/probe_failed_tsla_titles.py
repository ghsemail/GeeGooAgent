#!/usr/bin/env python3
import json
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def get_key() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-signal"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh["host"],
        username=ssh["user"],
        password=ssh.get("password"),
        timeout=60,
    )
    _, stdout, _ = client.exec_command(
        "grep '^GEEGOO_SIGNAL_ANALYZE_API_KEY=' /root/apps/GeeGooSignal/.env | cut -d= -f2-"
    )
    key = stdout.read().decode().strip()
    client.close()
    return key


def main() -> int:
    key = get_key()
    url = "http://146.56.225.252:3230/enrichStockNews"
    titles = [
        "How to Deploy $1000 Across Tesla and Ford for Growth and Optionality",
        (
            "Elon Musk's Boring Company Is Raising Money at a $20 Billion Valuation. "
            "Here's How His Empire Outside Tesla Is Growing."
        ),
        (
            "Elon Musk Owns 20% of Tesla, a Stake Worth Roughly $200 Billion. "
            "Here's Why His Ownership Level Matters for Shareholders."
        ),
    ]
    for i, title in enumerate(titles):
        body = json.dumps({"title": title, "snippet": ""}).encode()
        req = urllib.request.Request(
            url,
            data=body,
            headers={
                "Content-Type": "application/json",
                "Authorization": "Bearer " + key,
            },
        )
        try:
            resp = urllib.request.urlopen(req, timeout=180)
            data = json.loads(resp.read())
            print(
                i,
                "fallback=",
                data.get("used_fallback"),
                "cn_len=",
                len(data.get("title_cn") or ""),
                "model=",
                data.get("model"),
                "title_cn=",
                (data.get("title_cn") or "")[:60],
            )
        except Exception as exc:
            print(i, "ERROR", exc)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
