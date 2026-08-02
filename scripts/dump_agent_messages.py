#!/usr/bin/env python3
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from verify_agent_chat_e2e import BOT_KEY, USER, request, ssh_bot, REMOTE_ENSURE_MCP

c = ssh_bot()
_, o, e = c.exec_command(
    "python3 - <<'PY'\n" + REMOTE_ENSURE_MCP + "\nPY",
    timeout=30,
)
mcp = (o.read() + e.read()).decode().strip().splitlines()[-1].split(" ", 1)[-1]
c.close()

base = "http://118.195.135.97:3110"
_, body = request(
    base,
    "/op_agent/v1/sessions?limit=5",
    api_key=BOT_KEY,
    mcp_token=mcp,
    user_id=USER,
)
sessions = json.loads(body).get("sessions") or []
print("sessions", len(sessions))
for s in sessions[:3]:
    sid = s["id"]
    _, mb = request(
        base,
        f"/op_agent/v1/dashboard/sessions/{sid}/messages",
        api_key=BOT_KEY,
        mcp_token=mcp,
        user_id=USER,
    )
    msgs = json.loads(mb).get("messages") or []
    for m in msgs:
        if m.get("role") != "assistant":
            continue
        t = m.get("content") or ""
        print("===", sid, "len", len(t), "===")
        print(t[:3000])
        print()
        break
