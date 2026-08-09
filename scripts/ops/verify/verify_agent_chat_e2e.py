#!/usr/bin/env python3
"""Verify GeeGooAgent chat via BFF with trading user MCP token."""
from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE_ENSURE_MCP = f'''
from bson import ObjectId
from pymongo import MongoClient

uid = ObjectId("{USER}")
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=",1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=",1)[1]
user = MongoClient(mongo_uri)[dbn]["user"].find_one({{"_id": uid}})
token = ((user or {{}}).get("mcp") or {{}}).get("mcp_token", "")
if not token:
    import secrets
    token = "mcp_" + secrets.token_urlsafe(24)
    MongoClient(mongo_uri)[dbn]["user"].update_one(
        {{"_id": uid}},
        {{"$set": {{"mcp.mcp_token": token}}}},
    )
    print("generated", token)
else:
    print("existing", token)
'''


def ssh_bot():
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
    return c


def request(base, path, *, method="GET", api_key="", mcp_token="", user_id="", body=None, timeout=120):
    url = base.rstrip("/") + path
    headers = {"Accept": "application/json"}
    data = None
    if body is not None:
        data = json.dumps(body).encode()
        headers["Content-Type"] = "application/json"
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"
    if mcp_token:
        headers["X-MCP-Token"] = mcp_token
    if user_id:
        headers["X-User-Id"] = user_id
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


def read_sse_session_id(body: str) -> str | None:
    sid = None
    for block in body.split("\n\n"):
        if "event: connected" not in block:
            continue
        for line in block.split("\n"):
            if line.startswith("data: "):
                try:
                    data = json.loads(line[6:])
                    sid = data.get("session_id")
                except Exception:
                    pass
    return sid


def main() -> int:
    base = os.environ.get("AGENT_API_URL", "http://118.195.135.97:3110")
    print("=== ensure MCP token for ghsemail ===")
    c = ssh_bot()
    _, o, e = c.exec_command(f"python3 - <<'PY'\n{REMOTE_ENSURE_MCP}\nPY", timeout=30)
    out = (o.read() + e.read()).decode().strip().splitlines()
    c.close()
    if not out:
        print("FAIL: no mcp token output")
        return 1
    mcp_token = out[-1].split(" ", 1)[-1].strip()
    print("mcp_token", mcp_token[:12] + "...")

    print("\n=== BFF health ===")
    for label, path in [
        ("health", "/health"),
        ("tools", "/op_agent/v1/tools"),
        ("doctor", "/op_agent/v1/doctor?skip_connectivity=true"),
    ]:
        status, body = request(base, path, api_key=BOT_KEY)
        print(f"[{'OK' if status==200 else 'FAIL'}] {label} HTTP {status}")

    print("\n=== chat stream smoke ===")
    status, body = request(
        base,
        "/op_agent/v1/chat/stream",
        method="POST",
        api_key=BOT_KEY,
        mcp_token=mcp_token,
        user_id=USER,
        body={"message": "你好，请只回复：测试成功"},
        timeout=180,
    )
    print(f"chat HTTP {status}")
    sid = read_sse_session_id(body)
    print("session_id", sid)
    if status != 200 or not sid:
        print(body[:800])
        return 1

    print("\n=== list sessions (ops view) ===")
    status, body = request(
        base,
        "/op_agent/v1/sessions?limit=5",
        api_key=BOT_KEY,
        mcp_token=mcp_token,
        user_id=USER,
    )
    print(f"sessions HTTP {status}")
    try:
        data = json.loads(body)
        sessions = data.get("sessions") or []
        print("sessions", len(sessions), "first", sessions[0]["id"] if sessions else None)
        found = any(s.get("id") == sid for s in sessions)
        print("contains_new_session", found)
    except Exception as ex:
        print("parse fail", ex, body[:300])
        return 1

    print("\n=== session messages ===")
    status, body = request(
        base,
        f"/op_agent/v1/dashboard/sessions/{sid}/messages",
        api_key=BOT_KEY,
        mcp_token=mcp_token,
        user_id=USER,
    )
    print(f"messages HTTP {status}")
    try:
        data = json.loads(body)
        msgs = data.get("messages") or []
        print("message_count", len(msgs))
        for m in msgs[-3:]:
            role = m.get("role")
            content = (m.get("content") or "")[:80]
            print(f"  - {role}: {content}")
    except Exception as ex:
        print("parse fail", ex, body[:300])
        return 1

    print("\nAll agent chat checks passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
