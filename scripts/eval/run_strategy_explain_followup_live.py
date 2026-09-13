#!/usr/bin/env python3
"""Live smoke: strategy_signal_explain_followup (probe → diagnose explain).

Runs on geegoo-agent host against localhost :3400.
"""
from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

CASE_ID = "strategy_signal_explain_followup"
CHAT_TIMEOUT = int(os.environ.get("STRATEGY_EVAL_CHAT_TIMEOUT", "900"))
VERIFY_TIMEOUT = 120

TURN1 = (
    "我想看一下「RSI阈值信号（默认：RSI<30 买、RSI>70 卖，period=25）」"
    "在 Apple (AAPL.US) 上，最近有没有出现买卖信号？"
)
TURN2 = "为什么一个信号都没有？帮我从数据和规则上解释一下"


def load_env() -> tuple[str, str]:
    home = os.path.expanduser("~/.geegoo")
    env: dict[str, str] = {}
    with open(os.path.join(home, "agent.env"), encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            k = k.strip()
            if k.startswith("export "):
                k = k[7:].strip()
            env[k] = v.strip().strip('"').strip("'")
    cfg = json.load(open(os.path.join(home, "config.json"), encoding="utf-8"))
    runtime_key = env.get("GEEGOO_AGENT_RUNTIME_API_KEY", "") or cfg.get("api_key", "")
    mcp_token = cfg.get("mcp_token", "")
    if not mcp_token:
        raise RuntimeError("missing mcp_token in config.json")
    return runtime_key, mcp_token


def get_json(url: str, headers: dict, timeout: int) -> dict:
    req = urllib.request.Request(url, headers=headers, method="GET")
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8", errors="replace"))


def post_json(url: str, body: dict, headers: dict, timeout: int) -> tuple[int, str]:
    data = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


def chat_turn(runtime_key: str, mcp_token: str, message: str, session_id: str = "") -> dict:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-Approve-Writes": "true",
    }
    body = {"message": message, "mcp_token": mcp_token}
    if session_id:
        body["session_id"] = session_id
    status, raw = post_json("http://127.0.0.1:3400/v1/chat/stream", body, headers, CHAT_TIMEOUT)
    if status != 200:
        return {"ok": False, "error": f"chat HTTP {status}: {raw[:500]}"}

    sid = session_id
    turn_end: dict = {}
    event = ""
    for line in raw.splitlines():
        if line.startswith("event:"):
            event = line[6:].strip()
        elif line.startswith("data:"):
            payload = line[5:].strip()
            if event == "connected":
                try:
                    sid = json.loads(payload).get("session_id", sid) or sid
                except json.JSONDecodeError:
                    pass
            elif event == "turn_end":
                try:
                    turn_end = json.loads(payload)
                except json.JSONDecodeError:
                    turn_end = {"raw": payload}
    return {
        "ok": not bool(turn_end.get("failed")),
        "session_id": sid,
        "reply": turn_end.get("assistant_text") or "",
        "failed": bool(turn_end.get("failed")),
        "error": turn_end.get("error", ""),
    }


def resolve_clarify(runtime_key: str, session_id: str, choice: str) -> bool:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
    }
    status, raw = post_json(
        "http://127.0.0.1:3400/v1/chat/clarify",
        {"session_id": session_id, "choice": choice},
        headers,
        120,
    )
    return status == 200


def session_tools(runtime_key: str, session_id: str) -> list[str]:
    headers = {"Authorization": f"Bearer {runtime_key}"}
    trace = get_json(f"http://127.0.0.1:3400/v1/sessions/{session_id}/trace", headers, 60)
    tools: list[str] = []
    for rec in trace.get("step_records") or []:
        tool = (rec.get("tool_name") or rec.get("ToolName") or "").strip()
        if tool:
            tools.append(tool)
    return tools


def verify_case(runtime_key: str, session_id: str) -> dict:
    headers = {"Content-Type": "application/json", "Authorization": f"Bearer {runtime_key}"}
    status, raw = post_json(
        f"http://127.0.0.1:3400/v1/dashboard/eval/cases/{CASE_ID}/verify",
        {"session_id": session_id},
        headers,
        VERIFY_TIMEOUT,
    )
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return {"ok": False, "http": status, "raw": raw[:800]}
    data["http"] = status
    return data


def maybe_clarify(runtime_key: str, session_id: str, defaults: list[str]) -> None:
    headers = {"Authorization": f"Bearer {runtime_key}"}
    try:
        status = get_json(
            f"http://127.0.0.1:3400/v1/sessions/status?session_id={session_id}",
            headers,
            30,
        )
    except Exception:
        return
    pc = status.get("pending_clarify")
    if not pc:
        return
    choices = pc.get("choices") or []
    pick = ""
    for d in defaults:
        for c in choices:
            if d in c or c in d:
                pick = c
                break
        if pick:
            break
    if not pick and choices:
        pick = choices[0]
    if pick:
        print(f"  clarify -> {pick!r}", flush=True)
        resolve_clarify(runtime_key, session_id, pick)


def main() -> int:
    runtime_key, mcp_token = load_env()
    clarify_defaults = [
        "RSI阈值信号（默认：RSI<30 买、RSI>70 卖，period=25）",
        "RSI阈值信号",
        "AAPL.US",
        "Apple",
    ]

    print(f"=== {CASE_ID} turn 1 (probe) ===", flush=True)
    r1 = chat_turn(runtime_key, mcp_token, TURN1)
    if not r1.get("session_id"):
        print("FAIL turn1:", r1, flush=True)
        return 1
    sid = r1["session_id"]
    maybe_clarify(runtime_key, sid, clarify_defaults)
    if not r1.get("ok"):
        print("FAIL turn1:", r1.get("error") or r1, flush=True)
        return 1
    print(f"session={sid}", flush=True)
    print(f"reply: {(r1.get('reply') or '')[:240]}", flush=True)
    tools1 = session_tools(runtime_key, sid)
    print(f"tools after turn1: {tools1}", flush=True)

    print(f"\n=== {CASE_ID} turn 2 (explain) ===", flush=True)
    r2 = chat_turn(runtime_key, mcp_token, TURN2, sid)
    maybe_clarify(runtime_key, sid, clarify_defaults)
    if not r2.get("ok"):
        print("FAIL turn2:", r2.get("error") or r2, flush=True)
        return 1
    print(f"reply: {(r2.get('reply') or '')[:400]}", flush=True)
    tools = session_tools(runtime_key, sid)
    print(f"all tools: {tools}", flush=True)

    has_diagnose = "diagnose_bot_signal_series" in tools
    has_probe_turn2 = tools.count("probe_bot_signal_series") > 1
    print(f"diagnose_called={has_diagnose} extra_probe={has_probe_turn2}", flush=True)

    v = verify_case(runtime_key, sid)
    if v.get("http") == 200:
        print(f"\nverify ok={v.get('ok')} status={v.get('status')}", flush=True)
        print(f"detail: {v.get('detail', '')}", flush=True)
        for c in v.get("checks") or []:
            mark = "PASS" if c.get("passed") else "FAIL"
            print(f"  [{mark}] {c.get('type')}: {c.get('detail')}", flush=True)
        return 0 if v.get("ok") else 1

    print(f"verify HTTP {v.get('http')} (fallback manual checks)", flush=True)
    ok = has_diagnose and not has_probe_turn2 and len(r2.get("reply") or "") >= 80
    keywords = ["阈值", "RSI", "信号", "参数", "触发", "没"]
    reply = r2.get("reply") or ""
    kw_hit = sum(1 for k in keywords if k in reply)
    print(f"keyword_hits={kw_hit}/6 min_reply_ok={len(reply) >= 80}", flush=True)
    return 0 if ok and kw_hit >= 2 else 1


if __name__ == "__main__":
    raise SystemExit(main())
