#!/usr/bin/env python3
"""Increase nginx op_catalog/op_signal proxy_read_timeout to 180s."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    ssh_cfg = cfg["targets"]["geegoo-tradingsignal"]["ssh"]
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=60,
    )

    def run(cmd: str) -> str:
        _, o, e = client.exec_command(cmd)
        return (o.read() + e.read()).decode()

    nginx_id = run(
        "docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'"
    ).strip()
    if not nginx_id:
        print("nginx :8088 not found", file=sys.stderr)
        return 1

    conf_path = "/etc/nginx/conf.d/default.conf"
    conf = run(f"docker exec {nginx_id} cat {conf_path}")
    new_conf = conf.replace("proxy_read_timeout 120s", "proxy_read_timeout 180s")
    if new_conf == conf and "proxy_read_timeout 180s" not in conf:
        print("no proxy_read_timeout 120s to patch", file=sys.stderr)
        return 1

    remote_tmp = "/tmp/nginx-default.conf"
    sftp = client.open_sftp()
    with sftp.file(remote_tmp, "w") as f:
        f.write(new_conf)
    sftp.close()
    run(f"docker cp {remote_tmp} {nginx_id}:{conf_path}")
    test = run(f"docker exec {nginx_id} nginx -t")
    print(test.strip())
    if "successful" not in test:
        return 1
    run(f"docker exec {nginx_id} nginx -s reload")
    print("nginx reloaded with 180s proxy timeout")
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
