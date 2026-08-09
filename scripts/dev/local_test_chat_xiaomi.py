#!/usr/bin/env python3
"""Local chat SSE smoke: 查询小米股价 via agent BFF."""
from __future__ import annotations

import json
import sys
import time
import urllib.request
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")
BFF = "http://118.195.135.97:3110/op_agent/v1/chat/stream"
API_KEY = (
    "sk-7a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2"
)


def agent_token() -> str:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    s = cfg["targets"]["geegoo-agent"]["ssh"]
    c = paramiko.SSHClient()
    c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    c.connect(s["host"], username=s["user"], password=s.get("password"), timeout=30)
    _, o, _ = c.exec_command(
        "python3 -c \"import json; print(json.load(open('/home/ubuntu/.geegoo/config.json')).get('mcp_token',''))\"",
        timeout=30,
    )
    token = o.read().decode().strip()
    c.close()
    return token


def parse_events(raw: str) -> list[tuple[float, str, dict]]:
    events: list[tuple[float, str, dict]] = []
    t0 = time.time()
    event = ""
    for block in raw.split("\n\n"):
        if not block.strip():
            continue
        data_line = None
        for line in block.split("\n"):
            if line.startswith("event: "):
                event = line[7:].strip()
            elif line.startswith("data: "):
                data_line = line[6:]
        if data_line is None:
            continue
        try:
            data = json.loads(data_line)
        except json.JSONDecodeError:
            data = {"raw": data_line}
        events.append((round(time.time() - t0, 2), event or "message", data))
    return events


def main() -> int:
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    token = agent_token()
    body = json.dumps(
        {"message": "查询小米的股价", "session_id": "", "mcp_token": token},
        ensure_ascii=False,
    ).encode("utf-8")
    req = urllib.request.Request(
        BFF,
        data=body,
        method="POST",
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {API_KEY}",
            "X-MCP-Token": token,
            "X-Approve-Writes": "true",
        },
    )
    print("POST", BFF)
    with urllib.request.urlopen(req, timeout=180) as resp:
        raw = resp.read().decode("utf-8", errors="replace")
    events = parse_events(raw)
    print(f"events={len(events)}")
    for ts, ev, data in events:
        if ev in {
            "connected",
            "turn_start",
            "gate",
            "llm_start",
            "tool_start",
            "tool_done",
            "turn_end",
            "error",
        }:
            summary = {k: data[k] for k in ("name", "status", "summary", "decision") if k in data}
            print(f"{ts:6.2f}s  {ev:12}  {summary or data.get('message') or ''}")

    tool_done = [e for e in events if e[1] == "tool_done"]
    errors = [e for e in events if e[1] == "error"]
    price_tools = [
        e for e in tool_done if "current_price" in str(e[2].get("name", "")).lower()
    ]
    if errors:
        print("ERROR event:", errors[-1][2])
        return 1
    if not price_tools:
        print("WARN: no get_current_price tool_done seen")
    else:
        print("get_current_price:", price_tools[-1][2])
    turn_end = [e for e in events if e[1] == "turn_end"]
    if turn_end:
        text = turn_end[-1][2].get("assistant_text", "")
        print("reply:", (text or "")[:300])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
