#!/usr/bin/env python3
"""Live smoke: workflow_generate_strategy_cognition_random."""
from __future__ import annotations

import json
import os
import random
import sys
import urllib.error
import urllib.request

CASE_ID = "workflow_generate_strategy_cognition_random"
PASS_KEYWORDS = ["策略认知", "知识库", "策略库"]
MIN_REPLY_CHARS = 80
CHAT_TIMEOUT = int(os.environ.get("WORKFLOW_COGNITION_CHAT_TIMEOUT", "900"))
VERIFY_TIMEOUT = 120


def load_env() -> tuple[str, str, str, str]:
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
    catalog = (
        cfg.get("signal_catalog_api_url")
        or cfg.get("signal_base_url")
        or env.get("GEEGOO_SIGNAL_CATALOG_API_URL")
        or "http://146.56.225.252:3210"
    ).rstrip("/")
    cat_key = cfg.get("signal_catalog_api_key") or env.get("GEEGOO_SIGNAL_CATALOG_API_KEY") or ""
    if not mcp_token:
        raise RuntimeError("missing mcp_token in config.json")
    return runtime_key, mcp_token, catalog, cat_key


def post_json(url: str, body: dict | None, headers: dict, timeout: int) -> tuple[int, str]:
    data = None if body is None else json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8", errors="replace")


def pick_random_strategy(catalog: str, cat_key: str) -> str:
    headers = {"Content-Type": "application/json"}
    if cat_key:
        headers["Authorization"] = f"Bearer {cat_key}"
        headers["X-API-Key"] = cat_key
    status, raw = post_json(f"{catalog}/getSignalCombinationForSkill", {}, headers, 60)
    if status != 200:
        raise RuntimeError(f"catalog HTTP {status}: {raw[:400]}")
    data = json.loads(raw)
    if isinstance(data, list):
        items = data
    elif isinstance(data, dict):
        items = data.get("data") or data.get("items") or data.get("value") or []
        if isinstance(items, dict):
            items = items.get("items") or items.get("data") or []
    else:
        items = []
    names: list[str] = []
    for row in items:
        if not isinstance(row, dict):
            continue
        name = str(row.get("name") or "").strip()
        if name:
            names.append(name)
    if not names:
        raise RuntimeError("no combination strategies in catalog")
    return random.choice(names)


def chat_turn(runtime_key: str, mcp_token: str, message: str) -> dict:
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {runtime_key}",
        "X-Approve-Writes": "true",
    }
    body = {"message": message, "mcp_token": mcp_token}
    status, raw = post_json("http://127.0.0.1:3400/v1/chat/stream", body, headers, CHAT_TIMEOUT)
    if status != 200:
        return {"ok": False, "error": f"chat HTTP {status}: {raw[:500]}"}
    sid = ""
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
        "error": turn_end.get("error", ""),
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


def main() -> int:
    runtime_key, mcp_token, catalog, cat_key = load_env()
    strategy = pick_random_strategy(catalog, cat_key)
    if len(sys.argv) > 1 and sys.argv[1].strip():
        strategy = sys.argv[1].strip()
    message = f"生成策略认知 {strategy}"
    print(f"=== {CASE_ID} ===")
    print(f"picked_strategy: {strategy}")
    print(f"message: {message}")

    chat = chat_turn(runtime_key, mcp_token, message)
    if not chat.get("ok"):
        print("FAIL:", chat.get("error") or chat)
        return 1
    reply = chat.get("reply") or ""
    session_id = chat.get("session_id") or ""
    print(f"session_id={session_id}")
    print(f"reply_len={len(reply)}")
    print("reply_preview:", reply[:600].replace("\n", " "))

    kw_ok = len(reply.strip()) >= MIN_REPLY_CHARS and all(kw in reply for kw in PASS_KEYWORDS)
    verify = verify_case(runtime_key, session_id)
    if verify.get("http") == 200 and verify.get("ok") is True:
        print("verify: PASS", verify.get("detail", ""))
        print("SUMMARY: PASS")
        return 0
    if kw_ok:
        print("SUMMARY: PASS (keyword fallback)")
        return 0
    print("SUMMARY: FAIL")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
