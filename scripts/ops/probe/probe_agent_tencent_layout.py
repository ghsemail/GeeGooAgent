#!/usr/bin/env python3
"""Probe agent with stock analysis prompt and validate layout-friendly markdown."""
from __future__ import annotations

import json
import re
import sys
import urllib.error
import urllib.request
from pathlib import Path

# Reuse token helper from verify_agent_chat_e2e
sys.path.insert(0, str(Path(__file__).resolve().parent))
from verify_agent_chat_e2e import (  # noqa: E402
    BOT_KEY,
    USER,
    read_sse_session_id,
    request,
    ssh_bot,
    REMOTE_ENSURE_MCP,
)

PROMPT = "帮我分析腾讯股价"


def strip_think_tags(text: str) -> str:
    text = re.sub(
        r"<\s*redacted_thinking\s*>[\s\S]*?<\s*/\s*redacted_thinking\s*>",
        "",
        text,
        flags=re.I,
    )
    text = re.sub(r"<\s*redacted_thinking\s*>[\s\S]*$", "", text, flags=re.I)
    text = re.sub(
        r"<\s*think\s*>[\s\S]*?<\s*/\s*think\s*>",
        "",
        text,
        flags=re.I,
    )
    return text.strip()


def layout_checks(text: str) -> dict[str, bool]:
    has_heading = bool(re.search(r"^#{1,4}\s+", text, re.M))
    has_conclusion = bool(
        re.search(r"^#{1,4}\s*(结论|投资建议|操作建议|综合结论|总结|建议)", text, re.M)
    )
    has_pipe_table = "|---" in text or re.search(r"^\|.+\|.+\|$", text, re.M)
    has_kv_dot = bool(re.search(r"\d{4}-\d{2}-\d{2}\s*·", text))
    has_think_leak = bool(
        re.search(r"<\s*/?\s*(?:redacted_)?think(?:ing)?\s*>", text, re.I)
    )
    return {
        "has_heading": has_heading,
        "has_conclusion_section": has_conclusion,
        "no_raw_pipe_table": not has_pipe_table,
        "has_date_kv_or_content": has_kv_dot or len(text) > 200,
        "no_think_tag_leak": not has_think_leak,
        "non_empty": len(text) > 80,
    }


def extract_assistant_text(sse_body: str) -> str:
    chunks: list[str] = []
    for block in sse_body.split("\n\n"):
        if "event: message_delta" not in block and "event: stream_delta" not in block:
            continue
        for line in block.split("\n"):
            if not line.startswith("data: "):
                continue
            try:
                data = json.loads(line[6:])
            except json.JSONDecodeError:
                continue
            delta = data.get("delta") or data.get("content") or ""
            if isinstance(delta, str) and delta:
                chunks.append(delta)
    return "".join(chunks).strip()


def main() -> int:
    base = "http://118.195.135.97:3110"
    print("=== ensure MCP token ===")
    c = ssh_bot()
    _, o, e = c.exec_command(f"python3 - <<'PY'\n{REMOTE_ENSURE_MCP}\nPY", timeout=30)
    out = (o.read() + e.read()).decode().strip().splitlines()
    c.close()
    if not out:
        print("FAIL: no mcp token")
        return 1
    mcp_token = out[-1].split(" ", 1)[-1].strip()

    print(f"\n=== chat: {PROMPT!r} ===")
    status, body = request(
        base,
        "/op_agent/v1/chat/stream",
        method="POST",
        api_key=BOT_KEY,
        mcp_token=mcp_token,
        user_id=USER,
        body={"message": PROMPT},
        timeout=300,
    )
    print(f"HTTP {status}")
    sid = read_sse_session_id(body)
    text = extract_assistant_text(body)
    if status != 200:
        print(body[:1200])
        return 1

    print("session_id", sid)
    display_text = strip_think_tags(text)
    print("assistant_chars", len(display_text), f"(raw {len(text)})")
    print("\n--- preview (first 600 chars, stripped) ---")
    print(display_text[:600])
    print("\n--- layout checks (app-visible text) ---")
    checks = layout_checks(display_text)
    for k, v in checks.items():
        print(f"  [{'OK' if v else 'FAIL'}] {k}")

    if sid:
        st, msg_body = request(
            base,
            f"/op_agent/v1/dashboard/sessions/{sid}/messages",
            api_key=BOT_KEY,
            mcp_token=mcp_token,
            user_id=USER,
        )
        if st == 200:
            try:
                msgs = json.loads(msg_body).get("messages") or []
                assistant = [m for m in msgs if m.get("role") == "assistant"]
                if assistant:
                    stored = assistant[-1].get("content") or ""
                    stored_stripped = strip_think_tags(stored)
                    print("\n--- stored assistant (first 400, stripped) ---")
                    print(stored_stripped[:400])
                    stored_checks = layout_checks(stored_stripped)
                    print("--- stored layout checks ---")
                    for k, v in stored_checks.items():
                        print(f"  [{'OK' if v else 'FAIL'}] {k}")
                    checks = stored_checks
            except Exception as ex:
                print("stored parse warn", ex)

    ok = all(checks.values())
    print("\n" + ("PASS" if ok else "FAIL"))
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
