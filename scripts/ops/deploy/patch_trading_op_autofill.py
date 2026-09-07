#!/usr/bin/env python3
"""Deploy Chrome autofill bridge for trading_operation Flutter Web (:8088)."""
from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).resolve().parents[3]
BRIDGE_SRC = ROOT / "deploy" / "web" / "autofill-bridge.js"
SCRIPT_TAG = '<script src="autofill-bridge.js"></script>'
INDEX_MARK = "autofill-bridge.js"


def run(client: paramiko.SSHClient, cmd: str) -> tuple[str, str]:
    _, stdout, stderr = client.exec_command(cmd)
    return (
        stdout.read().decode("utf-8", "replace"),
        stderr.read().decode("utf-8", "replace"),
    )


def patch_index_html(html: str) -> str:
    if INDEX_MARK in html:
        return html
    needle = '<script src="clarify-overlay.js"></script>'
    if needle in html:
        return html.replace(
            needle,
            needle + "\n  " + SCRIPT_TAG,
            1,
        )
    needle = '<script src="flutter_bootstrap.js"'
    if needle in html:
        return html.replace(needle, "  " + SCRIPT_TAG + "\n  " + needle, 1)
    if "</body>" in html:
        return html.replace("</body>", f"  {SCRIPT_TAG}\n</body>", 1)
    return html + "\n" + SCRIPT_TAG + "\n"


def deploy(host: str, user: str, password: str, web_dir: str) -> int:
    if not BRIDGE_SRC.is_file():
        print(f"missing {BRIDGE_SRC}", file=sys.stderr)
        return 1

    bridge = BRIDGE_SRC.read_text(encoding="utf-8")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(host, username=user, password=password, timeout=30)

    sftp = client.open_sftp()
    remote_bridge = f"{web_dir}/autofill-bridge.js"
    remote_index = f"{web_dir}/index.html"
    remote_index_new = f"{web_dir}/index.html.autofill.bak"

    with sftp.file(remote_bridge, "w") as f:
        f.write(bridge)

    with sftp.file(remote_index, "r") as f:
        index_html = f.read().decode("utf-8", "replace")

    patched = patch_index_html(index_html)
    if patched != index_html:
        with sftp.file(remote_index_new, "w") as f:
            f.write(patched)
        run(client, f"mv {remote_index_new} {remote_index}")
        print("patched index.html")
    else:
        print("index.html already references autofill-bridge.js")

    sftp.close()

    out, err = run(
        client,
        "NGINX=$(docker ps --format '{{.ID}} {{.Ports}}' | awk '/8088->/{print $1; exit}'); "
        "echo nginx=$NGINX; "
        f"docker cp {web_dir}/autofill-bridge.js ${{NGINX}}:/usr/share/nginx/html/autofill-bridge.js; "
        f"docker cp {web_dir}/index.html ${{NGINX}}:/usr/share/nginx/html/index.html; "
        "docker exec $NGINX sh -c 'grep -n autofill-bridge /usr/share/nginx/html/index.html; "
        "ls -la /usr/share/nginx/html/autofill-bridge.js'",
    )
    if out.strip():
        print(out.rstrip())
    if err.strip():
        print("STDERR:", err.rstrip(), file=sys.stderr)

    client.close()
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--host",
        default=os.environ.get("TRADING_OP_SSH_HOST")
        or os.environ.get("GEEGOO_SIGNAL_SSH_HOST")
        or "146.56.225.252",
    )
    parser.add_argument(
        "--user",
        default=os.environ.get("TRADING_OP_SSH_USER")
        or os.environ.get("GEEGOO_SIGNAL_SSH_USER")
        or os.environ.get("GEEGOO_AGENT_SSH_USER")
        or "root",
    )
    parser.add_argument(
        "--password",
        default=os.environ.get("TRADING_OP_SSH_PASSWORD")
        or os.environ.get("GEEGOO_SIGNAL_SSH_PASSWORD")
        or os.environ.get("GEEGOO_AGENT_SSH_PASSWORD"),
    )
    parser.add_argument("--web-dir", default="/root/apps/trading_operation/web")
    args = parser.parse_args()

    if not args.password:
        print("missing SSH password (TRADING_OP_SSH_PASSWORD / GEEGOO_SIGNAL_SSH_PASSWORD)", file=sys.stderr)
        return 1

    return deploy(args.host, args.user, args.password, args.web_dir)


if __name__ == "__main__":
    raise SystemExit(main())
