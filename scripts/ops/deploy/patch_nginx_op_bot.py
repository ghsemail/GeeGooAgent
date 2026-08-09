#!/usr/bin/env python3
"""Add /op_bot nginx proxy on trading_operation host (8088 → GeeGooBot app-api :3100)."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

SKILL_DIR = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy")
DEPLOY_CFG = SKILL_DIR / "deploy.json"
BOT_UPSTREAM = "118.195.135.97:3100"
BOT_API_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"

SNIPPET_TEMPLATE = """
    location /op_bot/ {{
        proxy_pass http://{upstream}/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 180s;
    }}
"""


def run(client: paramiko.SSHClient, cmd: str) -> tuple[str, str]:
    _, stdout, stderr = client.exec_command(cmd)
    return stdout.read().decode(), stderr.read().decode()


def main() -> int:
    with DEPLOY_CFG.open(encoding="utf-8") as f:
        ssh_cfg = json.load(f)["targets"]["geegoo-tradingsignal"]["ssh"]

    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=60,
    )

    out, _ = run(
        client,
        "docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'",
    )
    nginx_id = out.strip()
    if not nginx_id:
        print("nginx container for :8088 not found", file=sys.stderr)
        return 1
    print(f"nginx_container={nginx_id}")

    conf_path = "/etc/nginx/conf.d/default.conf"
    conf, _ = run(client, f"docker exec {nginx_id} sh -c 'test -f {conf_path} && cat {conf_path}'")
    if not conf.strip():
        print("nginx config not found", file=sys.stderr)
        return 1

    snippet = SNIPPET_TEMPLATE.format(upstream=BOT_UPSTREAM)
    if "/op_bot/" in conf:
        print("op_bot location already present")
    else:
        marker = "    location / {"
        if marker not in conf:
            print("cannot find insertion point in nginx config", file=sys.stderr)
            return 1
        patched = conf.replace(marker, snippet + "\n" + marker, 1)
        remote_tmp = "/tmp/nginx-default.conf"
        sftp = client.open_sftp()
        with sftp.file(remote_tmp, "w") as f:
            f.write(patched)
        sftp.close()
        run(client, f"docker cp {remote_tmp} {nginx_id}:{conf_path}")
        test_out, test_err = run(client, f"docker exec {nginx_id} nginx -t")
        print(test_out.strip() or test_err.strip())
        if "successful" not in (test_out + test_err):
            return 1
        run(client, f"docker exec {nginx_id} nginx -s reload")
        print("nginx reloaded with /op_bot")

    verify_out, _ = run(
        client,
        f"curl -s -o /dev/null -w 'proxy_http=%{{http_code}}' "
        f"-H 'Authorization: Bearer {BOT_API_KEY}' "
        f"-H 'Content-Type: application/json' "
        f"-d '{{}}' "
        f"http://127.0.0.1:8088/op_bot/queryTradingDate",
    )
    print(verify_out.strip())
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
