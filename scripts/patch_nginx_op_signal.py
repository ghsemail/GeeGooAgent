#!/usr/bin/env python3
"""Add /op_signal nginx proxy on trading_operation host (8088 container)."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

SKILL_DIR = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy")
DEPLOY_CFG = SKILL_DIR / "deploy.json"

SNIPPET_TEMPLATE = """
    location /op_signal/ {{
        proxy_pass http://{upstream}/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 120s;
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

    conf_paths = [
        "/etc/nginx/conf.d/default.conf",
        "/etc/nginx/nginx.conf",
    ]
    conf_path = ""
    conf = ""
    for p in conf_paths:
        out, _ = run(client, f"docker exec {nginx_id} sh -c 'test -f {p} && cat {p}'")
        if out.strip():
            conf_path = p
            conf = out
            break
    if not conf:
        print("nginx config not found", file=sys.stderr)
        return 1
    print(f"config={conf_path}")

    upstream = "127.0.0.1:3200"
    for target in ("http://127.0.0.1:3200/health", "http://172.17.0.1:3200/health", "http://host.docker.internal:3200/health"):
        out, _ = run(client, f"docker exec {nginx_id} sh -c \"curl -sf -o /dev/null -w '%{{http_code}}' {target} || echo fail\"")
        code = out.strip()
        print(f"probe {target} -> {code}")
        if code.startswith("2"):
            upstream = target.replace("http://", "").replace("/health", "")
            break

    snippet = SNIPPET_TEMPLATE.format(upstream=upstream)

    if "/op_signal/" in conf:
        print("op_signal location already present")
    else:
        marker = "    location / {"
        if marker not in conf:
            print("cannot find insertion point in nginx config", file=sys.stderr)
            print(conf[:2000])
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
            print("nginx -t failed", file=sys.stderr)
            return 1
        run(client, f"docker exec {nginx_id} nginx -s reload")
        print("nginx reloaded")

    verify_out, _ = run(
        client,
        "curl -s -o /dev/null -w 'proxy_http=%{http_code}' "
        "-H 'Authorization: Bearer a76e2d4b4aa8b8eb154f3f2e8feff49a2c34c0f94d012402' "
        "http://127.0.0.1:8088/op_signal/v1/stocks/stats",
    )
    print(verify_out.strip())

    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
