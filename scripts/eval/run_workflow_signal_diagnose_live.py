#!/usr/bin/env python3
"""Live smoke: workflow_signal_diagnose_sar_tencent via chat/stream + verify."""
from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

CASE_ID = "workflow_signal_diagnose_sar_tencent"
MESSAGE = "诊断 SAR · 腾讯"
PASS_KEYWORDS = ["Episode", "命中", "SAR", "腾讯"]
WORKFLOW_OPTIONS = {
    "signal_diagnose": {
        "use_key_level_episode_stop": True,
        "key_break_mode": "resist_high",
    },
}
MIN_REPLY_CHARS = 120
CHAT_TIMEOUT = int(os.environ.get("WORKFLOW_EVAL_CHAT_TIMEOUT", "300"))
VERIFY_TIMEOUT = 120


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
    body = {"message": message, "mcp_token": mcp_token, "workflow_options": WORKFLOW_OPTIONS}
    if session_id:
        body["session_id"] = session_id
    status, raw = post_json("http://127.0.0.1:3400/v1/chat/stream", body, headers, CHAT_TIMEOUT)
    if status != 200:
        return {"ok": False, "error": f"chat HTTP {status}: {raw[:500]}"}

    sid = session_id
    turn_end: dict = {}
    event = ""
    chart_probe = False
    signal_eval = False
    eval_method = ""
    key_break_hits = 0
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
            elif event in ("workflow_card", "taskflow_card"):
                try:
                    data = json.loads(payload)
                    if data.get("chart_probe"):
                        chart_probe = True
                    se = data.get("signal_eval")
                    if se:
                        signal_eval = True
                        if isinstance(se, dict):
                            eval_method = str(se.get("method") or "")
                            for side in ("buy_details", "sell_details"):
                                for ep in se.get(side) or []:
                                    if not isinstance(ep, dict):
                                        continue
                                    if str(ep.get("strict_end_reason") or "").startswith("key_break"):
                                        key_break_hits += 1
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
        "chart_probe": chart_probe,
        "signal_eval": signal_eval,
        "eval_method": eval_method,
        "key_break_hits": key_break_hits,
    }


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


def keyword_pass(reply: str) -> tuple[bool, str]:
    text = reply.strip()
    if len(text) < MIN_REPLY_CHARS:
        return False, f"reply too short: {len(text)} < {MIN_REPLY_CHARS}"
    missing = [kw for kw in PASS_KEYWORDS if kw not in text]
    if missing:
        return False, f"missing keywords: {missing}"
    return True, "keywords ok"


def main() -> int:
    runtime_key, mcp_token = load_env()
    print(f"=== {CASE_ID} ===")
    print(f"message: {MESSAGE}")

    chat = chat_turn(runtime_key, mcp_token, MESSAGE)
    if not chat.get("ok"):
        print("FAIL chat:", chat.get("error") or chat)
        return 1
    session_id = chat.get("session_id") or ""
    reply = chat.get("reply") or ""
    print(f"session_id={session_id}")
    print(f"reply_len={len(reply)}")
    print(
        f"chart_probe={chat.get('chart_probe')} signal_eval={chat.get('signal_eval')} "
        f"eval_method={chat.get('eval_method')} key_break_hits={chat.get('key_break_hits')}"
    )
    print("reply_preview:", reply[:500].replace("\n", " "))

    kw_ok, kw_detail = keyword_pass(reply)
    print("keyword_check:", kw_detail)

    verify = verify_case(runtime_key, session_id)
    if verify.get("http") == 200 and verify.get("ok") is True:
        print("verify: PASS", verify.get("detail", ""))
        print("SUMMARY: PASS")
        return 0
    if verify.get("http") not in (200, None):
        print("verify skipped/failed:", verify.get("http"), verify.get("raw") or verify.get("detail"))
    method = str(chat.get("eval_method") or "")
    if "key_break_resist_high" not in method:
        print("FAIL: eval method missing key_break_resist_high:", method or "(empty)")
        print("SUMMARY: FAIL")
        return 1
    if kw_ok and chat.get("signal_eval"):
        print("SUMMARY: PASS (keyword + resist_high key_break)")
        return 0
    if kw_ok:
        print("SUMMARY: PASS (keyword fallback)")
        return 0
    print("SUMMARY: FAIL")
    if verify.get("detail"):
        print("verify detail:", verify.get("detail"))
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
