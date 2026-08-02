#!/usr/bin/env python3
"""Verify app-api 401 with wrong vs correct API key."""
from __future__ import annotations

import json
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
WRONG = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3g4h5"
RIGHT = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"


def main() -> None:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    ssh_cfg = cfg["targets"]["geegoo-bot"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=60,
    )

    def probe(key: str) -> str:
        _, o, e = client.exec_command(
            f"curl -s -o /dev/null -w '%{{http_code}}' "
            f"-H 'Authorization: Bearer {key}' "
            f"-H 'Content-Type: application/json' "
            f"-d '{{\"user_id\":\"test\"}}' "
            f"http://127.0.0.1:3100/getUserInfo"
        )
        return (o.read() + e.read()).decode().strip()

    print("wrong key:", probe(WRONG))
    print("right key:", probe(RIGHT))
    client.close()


if __name__ == "__main__":
    main()
