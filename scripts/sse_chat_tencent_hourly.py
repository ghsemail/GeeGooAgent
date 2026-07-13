#!/usr/bin/env python3
"""Test GeeGooAgent POST /v1/chat/stream SSE on production agent-runtime."""
from __future__ import annotations

import json
import re
import textwrap
from pathlib import Path

import paramiko

DEPLOY = Path(r"C:\Users\ghsemail\.cursor\skills\remote-deploy\deploy.json")

REMOTE_SCRIPT = r'''
import json, os, subprocess, sys, urllib.request

home = os.path.expanduser("~/.geegoo")
env = {}
with open(os.path.join(home, "agent.env"), encoding="utf-8") as f:
    for line in f:
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip().strip('"').strip("'")

runtime_key = env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "")
cfg = json.load(open(os.path.join(home, "config.json"), encoding="utf-8"))
mcp_token = cfg.get("mcp_token", "")

message = """请分析腾讯控股（00700.HK）今天的小时级价格走势与技术面。
步骤建议：
1. search_code 确认代码
2. get_current_price 获取最新价
3. get_mcp_analysis，name=腾讯控股，code=00700.HK，period=hourly，prompt_id 可用指数默认或技术面模板
请用中文简洁总结。"""

body = json.dumps({"message": message, "mcp_token": mcp_token}, ensure_ascii=False).encode("utf-8")
req = urllib.request.Request(
    "http://127.0.0.1:3400/v1/chat/stream",
    data=body,
    headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-Approve-Writes": "true",
    },
    method="POST",
)
try:
    with urllib.request.urlopen(req, timeout=300) as resp:
        raw = resp.read().decode("utf-8", errors="replace")
except Exception as e:
    print("REQUEST_ERROR:", e)
    sys.exit(1)

print(raw)
'''


def ssh_run(ssh_cfg: dict, cmd: str, timeout: int = 360) -> tuple[int, str, str]:
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    client.connect(
        hostname=ssh_cfg["host"],
        port=int(ssh_cfg.get("port", 22)),
        username=ssh_cfg["user"],
        password=ssh_cfg.get("password"),
        timeout=30,
    )
    _, stdout, stderr = client.exec_command(cmd, timeout=timeout)
    out = stdout.read().decode("utf-8", errors="replace")
    err = stderr.read().decode("utf-8", errors="replace")
    code = stdout.channel.recv_exit_status()
    client.close()
    return code, out, err


def parse_sse(raw: str) -> dict:
    events: list[tuple[str, str]] = []
    event = ""
    for line in raw.splitlines():
        if line.startswith("event:"):
            event = line[6:].strip()
        elif line.startswith("data:"):
            events.append((event, line[5:].strip()))
    progress = []
    turn_end = {}
    session_id = ""
    for name, data in events:
        if name == "connected":
            try:
                session_id = json.loads(data).get("session_id", "")
            except json.JSONDecodeError:
                pass
        elif name == "progress":
            try:
                progress.append(json.loads(data))
            except json.JSONDecodeError:
                progress.append({"raw": data})
        elif name == "turn_end":
            try:
                turn_end = json.loads(data)
            except json.JSONDecodeError:
                turn_end = {"raw": data}
    return {"session_id": session_id, "progress": progress, "turn_end": turn_end, "events": events}


def summarize_tools(progress: list[dict]) -> list[str]:
    lines = []
    for p in progress:
        data = p.get("data") if isinstance(p.get("data"), dict) else p
        if not isinstance(data, dict):
            continue
        ev = p.get("event") or data.get("event") or ""
        if ev in ("tool_start", "tool_end", "tool_result") or data.get("tool"):
            tool = data.get("tool") or data.get("tool_name") or data.get("name") or ""
            status = data.get("status") or data.get("tool_status") or ""
            summary = data.get("summary") or data.get("message") or ""
            if tool or summary:
                lines.append(f"  - {tool or ev}: {status} {summary}".strip())
    return lines


def main() -> int:
    cfg = json.loads(DEPLOY.read_text(encoding="utf-8"))
    agent = cfg["targets"]["geegoo-agent"]["ssh"]

    # upload inline python via base64
    import base64

    b64 = base64.b64encode(REMOTE_SCRIPT.encode()).decode()
    cmd = f"python3 -c \"import base64; exec(base64.b64decode('{b64}').decode())\""
    print("Calling POST /v1/chat/stream on agent-runtime :3400 ...")
    code, out, err = ssh_run(agent, cmd, timeout=360)
    if code != 0 and not out.strip():
        print("STDERR:", err)
        return code

    if "REQUEST_ERROR:" in out:
        print(out)
        return 1

    parsed = parse_sse(out)
    print(f"\nSession: {parsed['session_id']}")
    print(f"SSE events: {len(parsed['events'])}")

    tools = summarize_tools(parsed["progress"])
    if tools:
        print("\n工具调用:")
        for line in tools[:20]:
            print(line)
    else:
        # fallback: scan progress event names
        for p in parsed["progress"][:15]:
            inner = p.get("data", p)
            if isinstance(inner, dict):
                ev = p.get("event", "")
                print(f"  progress {ev}: {json.dumps(inner, ensure_ascii=False)[:200]}")

    te = parsed["turn_end"]
    print("\n--- 助手回复 ---")
    text = te.get("assistant_text") or te.get("raw") or ""
    if text:
        print(textwrap.fill(text, width=88))
    else:
        print("(无 assistant_text，原始 turn_end)")
        print(json.dumps(te, ensure_ascii=False, indent=2)[:2000])

    if te.get("failed"):
        print("\nTURN FAILED:", te.get("error"))
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
