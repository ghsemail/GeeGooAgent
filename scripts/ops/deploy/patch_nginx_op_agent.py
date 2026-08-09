#!/usr/bin/env python3
"""Add nginx /op_agent/ proxy to GeeGooBot agent-api BFF on bot host."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import paramiko

DEPLOY_CFG = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
BOT_BFF = "http://118.195.135.97:3110/op_agent/"

OP_AGENT_BLOCK = f"""
    location /op_agent/ {{
        if ($request_method = OPTIONS) {{
            add_header Access-Control-Allow-Origin *;
            add_header Access-Control-Allow-Methods 'GET, POST, PATCH, PUT, DELETE, OPTIONS';
            add_header Access-Control-Allow-Headers 'Authorization, Content-Type, X-MCP-Token, X-User-Id, X-Approve-Writes, X-Client-Source';
            return 204;
        }}
        add_header Access-Control-Allow-Origin * always;
        add_header Access-Control-Allow-Methods 'GET, POST, PATCH, PUT, DELETE, OPTIONS' always;
        add_header Access-Control-Allow-Headers 'Authorization, Content-Type, X-MCP-Token, X-User-Id, X-Approve-Writes, X-Client-Source' always;
        proxy_pass {BOT_BFF};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 320s;
        proxy_send_timeout 320s;
        proxy_buffering off;
        proxy_set_header Connection '';
        chunked_transfer_encoding off;
    }}
"""


def main() -> int:
    cfg = json.loads(DEPLOY_CFG.read_text(encoding="utf-8"))
    ssh_cfg = cfg["targets"]["geegoo-signal"]["ssh"]
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
    if "location /op_agent/" in conf:
        print("op_agent already configured")
    else:
        marker = "    location /op_catalog/ {"
        if marker not in conf:
            print("op_catalog block not found", file=sys.stderr)
            return 1
        new_conf = conf.replace(marker, OP_AGENT_BLOCK + "\n" + marker)
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
        print("nginx reloaded with /op_agent/")

    code = run("curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8088/op_agent/v1/tools")
    print("verify tools http:", code.strip())
    client.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
