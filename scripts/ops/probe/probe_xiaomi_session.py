#!/usr/bin/env python3
"""Inspect recent agent session about Xiaomi stock price."""
from __future__ import annotations

import json
import urllib.error
import urllib.request
from pathlib import Path

import paramiko

USER = "6366170502d5c175fd586fe8"
BOT_KEY = "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
BASE = "http://146.56.225.252:8088/op_agent"

REMOTE_MCP = """
from bson import ObjectId
from pymongo import MongoClient

uid = ObjectId("6366170502d5c175fd586fe8")
mongo_uri = "mongodb://127.0.0.1:27017"
dbn = "QT_DB"
for line in open("/home/ubuntu/apps/GeeGooBot/.env"):
    if line.startswith("GEEGOO_BOT_MONGO_URI="):
        mongo_uri = line.strip().split("=", 1)[1]
    if line.startswith("GEEGOO_BOT_MONGO_DB="):
        dbn = line.strip().split("=", 1)[1]
user = MongoClient(mongo_uri)[dbn]["user"].find_one({"_id": uid})
print(((user or {}).get("mcp") or {}).get("mcp_token", ""))
"""


def get_mcp_token() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    ssh = cfg["targets"]["geegoo-bot"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(ssh["host"], username=ssh["user"], password=ssh["password"], timeout=30)
    _, o, _ = c.exec_command(f"python3 - <<'PY'\n{REMOTE_MCP}\nPY", timeout=30)
    token = o.read().decode().strip()
    c.close()
    return token


def req(path: str, mcp: str) -> dict:
    url = BASE + path
    headers = {
        "Authorization": f"Bearer {BOT_KEY}",
        "X-User-Id": USER,
        "X-MCP-Token": mcp,
        "X-Client-Source": "trading_app",
    }
    r = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(r, timeout=60) as resp:
        return json.loads(resp.read().decode())


def main() -> None:
    mcp = get_mcp_token()
    print("mcp", mcp[:16] + "...")

    sessions = req("/v1/sessions?limit=25", mcp).get("sessions", [])
    print("\n=== recent sessions ===")
    for s in sessions[:15]:
        print(
            s.get("id"),
            "|",
            s.get("source"),
            "|",
            (s.get("title") or "")[:60],
        )

    hits: list[str] = []
    for s in sessions:
        blob = json.dumps(s, ensure_ascii=False)
        if any(k in blob for k in ("小米", "1810", "Xiaomi", "xiaomi", "股价")):
            hits.append(s["id"])
    trading_app = next((s["id"] for s in sessions if s.get("source") == "trading_app"), None)
    if trading_app and trading_app not in hits:
        hits.insert(0, trading_app)
    if not hits and sessions:
        hits = [sessions[0]["id"]]

    out_path = Path(__file__).with_name("probe_xiaomi_session.json")
    payload = {"sessions": sessions, "details": []}
    for sid in hits[:3]:
        detail = {"session_id": sid, "messages": req(f"/v1/dashboard/sessions/{sid}/messages", mcp)}
        try:
            detail["trace"] = req(f"/v1/sessions/{sid}/trace", mcp)
        except urllib.error.HTTPError as e:
            detail["trace_error"] = {"code": e.code, "body": e.read().decode(errors="replace")[:500]}
        payload["details"].append(detail)
    out_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2), encoding="utf-8")
    print("wrote", out_path)

    for sid in hits[:3]:
        print(f"\n===== SESSION {sid} =====")
        data = req(f"/v1/dashboard/sessions/{sid}/messages", mcp)
        print("title:", data.get("title"))
        for m in data.get("messages", []):
            role = m.get("role")
            content = m.get("content") or ""
            reasoning = m.get("reasoning_content") or ""
            print(
                "---",
                role,
                "content_len",
                len(content),
                "reasoning_len",
                len(reasoning),
                "tools",
                m.get("tool_call_count"),
            )
            if role == "user":
                print(content[:300])
            else:
                print("CONTENT (first 1200 chars):")
                print(content[:1200])
                if reasoning:
                    print("REASONING (first 600 chars):")
                    print(reasoning[:600])
                if len(content) > 1200:
                    print("... [truncated]")
        try:
            tr = req(f"/v1/sessions/{sid}/trace", mcp)
            steps = tr.get("steps") or tr.get("step_records") or tr.get("records") or []
            print("trace steps:", len(steps))
            for st in steps[:15]:
                if not isinstance(st, dict):
                    continue
                kind = st.get("kind") or st.get("type")
                summary = (st.get("summary") or st.get("content") or st.get("tool_name") or "")[:180]
                print(" ", kind, summary)
        except urllib.error.HTTPError as e:
            print("trace HTTP", e.code, e.read()[:300].decode(errors="replace"))


if __name__ == "__main__":
    main()
