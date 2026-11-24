"""Robust Agent runtime chat/stream client for macro batch (SSE + session recovery)."""

from __future__ import annotations

import json
import os
import socket
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from http.client import IncompleteRead
from typing import Any

# Pool/overall JEV synthesis often exceeds 6m; align with diagnose-detail batch budget (900s).
CHAT_TIMEOUT = int(os.environ.get("MACRO_BATCH_CHAT_TIMEOUT", "900"))
RECOVER_POLL_SEC = int(os.environ.get("MACRO_BATCH_CHAT_RECOVER_POLL_SEC", "5"))
RECOVER_MAX_SEC = int(os.environ.get("MACRO_BATCH_CHAT_RECOVER_MAX_SEC", "180"))
RESUME_ON_STALE = os.environ.get("MACRO_BATCH_CHAT_RESUME_ON_STALE", "1").strip().lower() not in (
    "0",
    "false",
    "no",
)


@dataclass
class ChatTurnResult:
    ok: bool
    session_id: str = ""
    run_id: str = ""
    reply: str = ""
    turn_end: dict[str, Any] = field(default_factory=dict)
    recovered: bool = False
    resumed: bool = False
    stream_error: str = ""
    events_seen: list[str] = field(default_factory=list)


def _runtime_headers(
    runtime_key: str,
    mcp_token: str,
    *,
    approve_writes: bool = True,
) -> dict[str, str]:
    h = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-MCP-Token": mcp_token,
    }
    if approve_writes:
        h["X-Approve-Writes"] = "true"
    return h


def parse_sse_payload(raw: str) -> tuple[str, str, dict[str, Any], list[str]]:
    """Parse a full or partial SSE body into session_id, run_id, turn_end, and event names."""
    sid = ""
    run_id = ""
    turn_end: dict[str, Any] = {}
    events: list[str] = []
    event = ""
    for line in raw.splitlines():
        if line.startswith("event:"):
            event = line[6:].strip()
            if event:
                events.append(event)
        elif line.startswith("data:"):
            payload = line[5:].strip()
            if not payload:
                continue
            try:
                data = json.loads(payload)
            except json.JSONDecodeError:
                continue
            if event == "connected":
                sid = str(data.get("session_id") or sid) or sid
            elif event == "signal_diagnose_result_ready":
                run_id = str(data.get("run_id") or run_id) or run_id
            elif event == "turn_end":
                turn_end = data
                sid = str(data.get("session_id") or sid) or sid
            elif event == "done":
                sid = str(data.get("session_id") or sid) or sid
    return sid, run_id, turn_end, events


def _consume_sse_stream(
    resp: Any,
    *,
    timeout_deadline: float,
) -> tuple[str, str, dict[str, Any], list[str], str]:
    """Read response incrementally; return parsed fields and optional transport error."""
    chunks: list[bytes] = []
    stream_err = ""
    try:
        while True:
            if time.monotonic() > timeout_deadline:
                stream_err = f"read timeout after {CHAT_TIMEOUT}s"
                break
            try:
                part = resp.read(65536)
            except IncompleteRead as exc:
                if exc.partial:
                    chunks.append(exc.partial)
                stream_err = f"IncompleteRead({len(b''.join(chunks))} bytes)"
                break
            except socket.timeout:
                stream_err = f"socket timeout after {CHAT_TIMEOUT}s"
                break
            if not part:
                break
            chunks.append(part)
    finally:
        try:
            resp.close()
        except Exception:
            pass
    raw = b"".join(chunks).decode("utf-8", errors="replace")
    sid, run_id, turn_end, events = parse_sse_payload(raw)
    return sid, run_id, turn_end, events, stream_err


def http_get_json(
    url: str,
    headers: dict[str, str],
    *,
    timeout: int = 60,
) -> tuple[int, Any, str]:
    req = urllib.request.Request(url, headers=headers, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            return resp.status, json.loads(raw) if raw.strip() else {}, raw
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            parsed = json.loads(raw) if raw.strip() else {}
        except json.JSONDecodeError:
            parsed = {"raw": raw}
        return e.code, parsed, raw


def fetch_session_status(
    agent_base: str,
    runtime_key: str,
    session_id: str,
) -> tuple[bool, dict[str, Any], str]:
    q = urllib.request.quote(session_id, safe="")
    url = f"{agent_base.rstrip('/')}/v1/sessions/status?session_id={q}"
    status, data, raw = http_get_json(url, _runtime_headers(runtime_key, "", approve_writes=False), timeout=60)
    if status != 200:
        return False, {}, f"status HTTP {status}: {raw[:300]}"
    if isinstance(data, dict) and isinstance(data.get("data"), dict):
        data = data["data"]
    return True, data if isinstance(data, dict) else {}, ""


def last_assistant_text(status: dict[str, Any]) -> str:
    messages = status.get("messages") or []
    for row in reversed(messages):
        if not isinstance(row, dict):
            continue
        if str(row.get("role") or "").lower() != "assistant":
            continue
        text = str(row.get("content") or "").strip()
        if text:
            return text
    return ""


def session_turn_complete(status: dict[str, Any], *, min_reply_len: int = 1) -> bool:
    if status.get("pending_clarify"):
        return False
    busy = bool(status.get("busy"))
    live = status.get("live") or {}
    if busy or (isinstance(live, dict) and live.get("busy")):
        return False
    reply = last_assistant_text(status)
    return len(reply) >= min_reply_len


def wait_session_turn(
    agent_base: str,
    runtime_key: str,
    session_id: str,
    *,
    min_reply_len: int = 20,
    max_wait_sec: int | None = None,
    poll_sec: int | None = None,
) -> tuple[bool, dict[str, Any], str]:
    if not session_id:
        return False, {}, "no session_id"
    deadline = time.monotonic() + (max_wait_sec if max_wait_sec is not None else RECOVER_MAX_SEC)
    interval = poll_sec if poll_sec is not None else RECOVER_POLL_SEC
    last_err = ""
    last_payload: dict[str, Any] = {}
    while time.monotonic() < deadline:
        ok, payload, err = fetch_session_status(agent_base, runtime_key, session_id)
        if not ok:
            last_err = err
            time.sleep(interval)
            continue
        last_payload = payload
        if session_turn_complete(payload, min_reply_len=min_reply_len):
            return True, payload, ""
        if payload.get("pending_clarify"):
            return False, payload, "session waiting for clarify"
        time.sleep(interval)
    return False, last_payload, last_err or "recover wait timeout"


def post_workflow_resume(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    *,
    mode: str = "continue",
) -> tuple[bool, dict[str, Any], str]:
    url = f"{agent_base.rstrip('/')}/v1/chat/workflow/resume"
    body = {"session_id": session_id, "mcp_token": mcp_token, "mode": mode}
    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        headers=_runtime_headers(runtime_key, mcp_token),
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=CHAT_TIMEOUT) as resp:
            raw = resp.read().decode("utf-8", errors="replace")
            parsed = json.loads(raw) if raw.strip() else {}
            if resp.status != 200:
                return False, parsed, f"resume HTTP {resp.status}"
            return True, parsed, ""
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            parsed = json.loads(raw) if raw.strip() else {}
        except json.JSONDecodeError:
            parsed = {"raw": raw}
        return False, parsed, f"resume HTTP {e.code}: {raw[:240]}"


def stream_chat_turn(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    message: str,
    *,
    body_extra: dict[str, Any] | None = None,
    session_id: str = "",
    recover: bool = True,
    min_reply_len: int = 20,
) -> ChatTurnResult:
    """One chat turn: POST /v1/chat/stream with SSE parse + optional session recovery."""
    body: dict[str, Any] = {"message": message, "mcp_token": mcp_token}
    if body_extra:
        body.update(body_extra)
    if session_id:
        body["session_id"] = session_id
    url = f"{agent_base.rstrip('/')}/v1/chat/stream"
    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        headers=_runtime_headers(runtime_key, mcp_token),
        method="POST",
    )
    out = ChatTurnResult(ok=False)
    try:
        resp = urllib.request.urlopen(req, timeout=CHAT_TIMEOUT)
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        out.stream_error = f"chat/stream HTTP {e.code}: {raw[:400]}"
        return out
    except urllib.error.URLError as e:
        out.stream_error = f"chat/stream URLError: {e}"
        return out

    deadline = time.monotonic() + CHAT_TIMEOUT + 5
    sid, run_id, turn_end, events, stream_err = _consume_sse_stream(resp, timeout_deadline=deadline)
    out.session_id = sid
    out.run_id = run_id
    out.turn_end = turn_end
    out.events_seen = events
    out.stream_error = stream_err

    failed = bool(turn_end.get("failed"))
    err = str(turn_end.get("error") or "")
    reply = str(turn_end.get("assistant_text") or "").strip()
    if turn_end and not failed and not err:
        out.ok = True
        out.reply = reply
        return out

    if recover and sid:
        ok, status, rec_err = wait_session_turn(
            agent_base,
            runtime_key,
            sid,
            min_reply_len=min_reply_len,
        )
        if ok:
            out.recovered = True
            out.reply = last_assistant_text(status)
            out.session_id = str(status.get("session_id") or sid)
            out.ok = len(out.reply) >= min_reply_len
            if out.ok:
                return out
        out.stream_error = stream_err or rec_err or "turn incomplete after recover"

        if RESUME_ON_STALE and sid:
            r_ok, r_body, r_err = post_workflow_resume(agent_base, runtime_key, mcp_token, sid)
            if r_ok:
                out.resumed = True
                text = str(r_body.get("assistant_text") or "").strip()
                if text and len(text) >= min_reply_len and not r_body.get("failed"):
                    out.ok = True
                    out.reply = text
                    out.session_id = str(r_body.get("session_id") or sid)
                    return out
            if r_err and "no active flow" not in r_err:
                out.stream_error = f"{out.stream_error}; resume: {r_err}"
    elif stream_err and not turn_end:
        out.stream_error = stream_err

    if failed or err:
        out.stream_error = out.stream_error or err or "turn failed"
    return out


def result_from_stream_error(result: ChatTurnResult) -> dict[str, Any]:
    """Shape compatible with legacy chat_diagnose dict."""
    if result.ok:
        return {
            "ok": True,
            "session_id": result.session_id,
            "run_id": result.run_id,
            "reply": result.reply,
            "recovered": result.recovered,
            "resumed": result.resumed,
        }
    err = result.stream_error or "chat failed"
    if result.recovered:
        err = f"{err} (recovered partial)"
    return {
        "ok": False,
        "session_id": result.session_id,
        "run_id": result.run_id,
        "error": err,
        "recovered": result.recovered,
        "resumed": result.resumed,
    }


def recover_turn_from_session(
    agent_base: str,
    runtime_key: str,
    mcp_token: str,
    session_id: str,
    *,
    min_reply_len: int = 20,
    try_resume: bool = True,
) -> ChatTurnResult:
    """Poll persisted session (and optional workflow resume) without a new chat/stream."""
    out = ChatTurnResult(ok=False, session_id=session_id)
    ok, status, rec_err = wait_session_turn(
        agent_base, runtime_key, session_id, min_reply_len=min_reply_len, max_wait_sec=30
    )
    if ok:
        out.recovered = True
        out.reply = last_assistant_text(status)
        out.ok = len(out.reply) >= min_reply_len
        return out
    if try_resume and RESUME_ON_STALE:
        r_ok, r_body, r_err = post_workflow_resume(agent_base, runtime_key, mcp_token, session_id)
        if r_ok:
            out.resumed = True
            text = str(r_body.get("assistant_text") or "").strip()
            if text and len(text) >= min_reply_len and not r_body.get("failed"):
                out.ok = True
                out.reply = text
                return out
        if r_err and "no active flow" not in r_err:
            out.stream_error = r_err
    out.stream_error = rec_err or out.stream_error or "session not ready"
    ok2, status2, _ = fetch_session_status(agent_base, runtime_key, session_id)
    if ok2:
        text = last_assistant_text(status2)
        if len(text) >= min_reply_len:
            out.recovered = True
            out.reply = text
            out.ok = True
    return out