#!/usr/bin/env python3
"""Look up QT_DB.user MCP token by App username (Cloud Agent / ops).

Uses remote-deploy SSH to GeeGooBot host and reads Mongo — not ~/.geegoo/config.json
on the agent VM (that file is usually the deploy operator's token, e.g. ghsemail).

Usage:
  python3 .cursor/scripts/lookup_user_mcp_token.py duoduo
  python3 .cursor/scripts/lookup_user_mcp_token.py duoduo --verify
"""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

REMOTE_SCRIPT = r"""
import json, os, subprocess, sys
from urllib.parse import urlparse
try:
    from pymongo import MongoClient
except ImportError:
    print(json.dumps({"error": "pymongo missing on bot host"}))
    sys.exit(1)

username = os.environ["LOOKUP_USERNAME"]
verify = os.environ.get("LOOKUP_VERIFY") == "1"

for path in ("/home/ubuntu/apps/GeeGooBot/.env", "/home/ubuntu/apps/TradingBot/.env"):
    try:
        with open(path) as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                k, v = line.split("=", 1)
                os.environ.setdefault(k.strip(), v.strip().strip('"').strip("'"))
    except FileNotFoundError:
        pass

uri = os.environ.get("GEEGOO_BOT_MONGO_URI", "")
db_name = os.environ.get("GEEGOO_BOT_MONGO_DB") or "QT_DB"
if not uri:
    print(json.dumps({"error": "GEEGOO_BOT_MONGO_URI not set on bot host"}))
    sys.exit(2)

client = MongoClient(uri, serverSelectionTimeoutMS=8000)
db = client[db_name]
doc = db.user.find_one({"username": username}, {"_id": 1, "username": 1, "mcp": 1})
if not doc:
    print(json.dumps({"error": "user not found", "username": username}))
    sys.exit(3)

mcp = doc.get("mcp") or {}
token = (mcp.get("mcp_token") or "").strip()
out = {
    "username": doc.get("username"),
    "user_id": str(doc["_id"]),
    "mcp_enabled": bool(mcp.get("enabled")),
    "mcp_token": token,
}
if verify and token:
    key = (
        os.environ.get("GEEGOO_BOT_SERVICE_API_KEY")
        or os.environ.get("GEEGOO_BOT_MCP_API_KEY")
        or os.environ.get("GEEGOO_BOT_APP_API_KEY")
        or ""
    )
    cmd = [
        "curl", "-sS", "-m", "20", "-X", "POST",
        "http://127.0.0.1:3120/checkTradingDay",
        "-H", "Content-Type: application/json",
        "-d", json.dumps({"mcp_token": token, "code": "00700.HK"}),
    ]
    if key:
        cmd.extend(["-H", f"Authorization: Bearer {key}"])
    p = subprocess.run(cmd, capture_output=True, text=True)
    try:
        out["mcp_verify"] = json.loads(p.stdout or "{}")
    except json.JSONDecodeError:
        out["mcp_verify_raw"] = (p.stdout or p.stderr or "").strip()[:300]
print(json.dumps(out, ensure_ascii=False))
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Lookup App user MCP token via GeeGooBot Mongo")
    parser.add_argument("username", help="QT_DB.user username (e.g. duoduo)")
    parser.add_argument(
        "--verify",
        action="store_true",
        help="Call mcp-api checkTradingDay on bot host with this token",
    )
    parser.add_argument(
        "--target",
        default="geegoo-bot",
        help="remote-deploy target (default geegoo-bot)",
    )
    args = parser.parse_args()

    skill_scripts = Path.home() / ".cursor/skills/remote-deploy/scripts"
    if not skill_scripts.is_dir():
        print("remote-deploy skill not installed; run .cursor/install-remote-deploy.sh", file=sys.stderr)
        return 1

    sys.path.insert(0, str(skill_scripts))
    from credentials import resolve_ssh_cfg  # noqa: E402

    import paramiko  # noqa: E402

    deploy_path = Path.home() / ".cursor/skills/remote-deploy/deploy.json"
    deploy = json.loads(deploy_path.read_text(encoding="utf-8"))
    targets = deploy.get("targets") or {}
    if args.target not in targets:
        print(f"unknown target {args.target}", file=sys.stderr)
        return 1

    spec = targets[args.target]
    ssh_cfg = resolve_ssh_cfg(spec["ssh"], target_name=args.target)

    with tempfile.NamedTemporaryFile("w", suffix=".py", delete=False) as tmp:
        tmp.write(REMOTE_SCRIPT)
        local_remote = tmp.name

    remote_path = "/tmp/lookup_user_mcp_token.py"
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    sftp = client.open_sftp()
    sftp.put(local_remote, remote_path)
    sftp.close()
    os.unlink(local_remote)

    env_prefix = f"LOOKUP_USERNAME={args.username!s} LOOKUP_VERIFY={'1' if args.verify else '0'}"
    cmd = f"bash -lc 'set -a; [ -f /home/ubuntu/apps/GeeGooBot/.env ] && . /home/ubuntu/apps/GeeGooBot/.env; set +a; {env_prefix} python3 {remote_path}'"
    _, stdout, stderr = client.exec_command(cmd, timeout=120)
    out = stdout.read().decode().strip()
    err = stderr.read().decode().strip()
    client.exec_command(f"rm -f {remote_path}")
    client.close()

    if err:
        print(err, file=sys.stderr)
    if not out:
        return 1
    try:
        payload = json.loads(out)
    except json.JSONDecodeError:
        print(out)
        return 1
    if payload.get("error"):
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return 1
    print(json.dumps(payload, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
